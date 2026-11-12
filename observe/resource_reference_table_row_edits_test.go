package observe

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

const fakeReferenceTableID = "41000001"

// fakeReferenceTableAPI serves one reference table. Like Observe, it stores the
// MD5 of each uploaded file as the checksum and clears it when rows are edited.
type fakeReferenceTableAPI struct {
	mu               sync.Mutex
	checksum         string
	uploads          int
	metadataPatches  int
	rejectUpdates    bool
	unexpectedRoutes []string
}

func (f *fakeReferenceTableAPI) editRows() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checksum = ""
}

func (f *fakeReferenceTableAPI) setRejectUpdates(reject bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejectUpdates = reject
}

func (f *fakeReferenceTableAPI) counts() (uploads, metadataPatches int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uploads, f.metadataPatches
}

func (f *fakeReferenceTableAPI) storeUpload(r *http.Request) error {
	file, _, err := r.FormFile("upload")
	if err != nil {
		return err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	f.checksum = hex.EncodeToString(hash.Sum(nil))
	f.uploads++
	return nil
}

func (f *fakeReferenceTableAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	tablePath := "/v1/referencetables/" + fakeReferenceTableID
	switch {
	case f.rejectUpdates && r.URL.Path == tablePath && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		http.Error(w, `{"message":"rejected update"}`, http.StatusBadRequest)
		return
	case r.URL.Path == "/v1/meta":
		_, _ = w.Write([]byte(`{"data":{"dataset":{"id":"41000002","name":"Reference Tables/hosts","typedef":{"def":{"fields":[
			{"name":"host","type":{"rep":"string"}},{"name":"owner","type":{"rep":"string"}}]}}}}}`))
		return
	case r.URL.Path == "/v1/referencetables" && r.Method == http.MethodPost,
		r.URL.Path == tablePath && r.Method == http.MethodPut:
		if err := f.storeUpload(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case r.URL.Path == tablePath && r.Method == http.MethodPatch:
		f.metadataPatches++
	case r.URL.Path == tablePath && r.Method == http.MethodGet:
	case r.URL.Path == tablePath && r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		f.unexpectedRoutes = append(f.unexpectedRoutes, r.Method+" "+r.URL.Path)
		http.Error(w, "unexpected route", http.StatusNotFound)
		return
	}
	_, _ = fmt.Fprintf(w, `{"id":%q,"label":"hosts","checksum":%q,"datasetId":"41000002","primaryKey":["host"],"labelField":"owner"}`,
		fakeReferenceTableID, f.checksum)
}

func newFakeReferenceTableClient(t *testing.T, api *fakeReferenceTableAPI) *observeclient.Client {
	t.Helper()
	server := httptest.NewServer(api)
	t.Cleanup(func() {
		server.Close()
		if len(api.unexpectedRoutes) > 0 {
			t.Errorf("unexpected requests: %v", api.unexpectedRoutes)
		}
	})
	metaAPI, err := meta.New(server.URL+"/v1/meta", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return &observeclient.Client{Config: &observeclient.Config{}, Rest: rest.New(server.URL, server.Client()), Meta: metaAPI}
}

func TestResourceReferenceTableReadChecksumAfterRowEdits(t *testing.T) {
	const stateChecksum = "9b2cf535f27731c974343645a3985328"
	for _, tc := range []struct {
		name          string
		allowRowEdits bool
		apiChecksum   string
		wantChecksum  string
	}{
		{"edited rows, row edits allowed", true, "", stateChecksum},
		{"edited rows, row edits not allowed", false, "", ""},
		{"new upload, row edits allowed", true, "891217caed9a1c2b325f23f418afbde5", "891217caed9a1c2b325f23f418afbde5"},
		{"unchanged, row edits not allowed", false, stateChecksum, stateChecksum},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeReferenceTableClient(t, &fakeReferenceTableAPI{checksum: tc.apiChecksum})
			data := resourceReferenceTable().Data(&terraform.InstanceState{
				ID: fakeReferenceTableID,
				Attributes: map[string]string{
					"checksum":        stateChecksum,
					"allow_row_edits": fmt.Sprint(tc.allowRowEdits),
				},
			})
			if diags := resourceReferenceTableRead(context.Background(), data, client); diags.HasError() {
				t.Fatalf("read reference table: %v", diags)
			}
			if got := data.Get("checksum"); got != tc.wantChecksum {
				t.Errorf("checksum = %q, want %q", got, tc.wantChecksum)
			}
		})
	}
}

func referenceTableRowEditsConfig(sourceFile string, allowRowEdits bool) string {
	return fmt.Sprintf(`
resource "observe_reference_table" "hosts" {
  label           = "hosts"
  source_file     = %[1]q
  checksum        = filemd5(%[1]q)
  primary_key     = ["host"]
  label_field     = "owner"
  allow_row_edits = %[2]t
}
`, sourceFile, allowRowEdits)
}

func writeReferenceTableCSV(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func checkReferenceTableAPICalls(api *fakeReferenceTableAPI, wantUploads, wantMetadataPatches int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		uploads, metadataPatches := api.counts()
		if uploads != wantUploads || metadataPatches != wantMetadataPatches {
			return fmt.Errorf("uploads = %d, metadata patches = %d, want %d and %d", uploads, metadataPatches, wantUploads, wantMetadataPatches)
		}
		return nil
	}
}

// testReferenceTableWithFakeAPI runs real Terraform plans and applies against api.
func testReferenceTableWithFakeAPI(t *testing.T, api *fakeReferenceTableAPI, steps []resource.TestStep) {
	client := newFakeReferenceTableClient(t, api)
	t.Setenv("OBSERVE_CUSTOMER", "1")
	t.Setenv("OBSERVE_API_TOKEN", "unused")
	// UnitTest builds the provider from the process environment. Blank every
	// login variable that ConflictsWith api_token so an ambient CI identity
	// (user_email/password or oauth2) cannot make PrepareProviderConfig fail.
	for _, name := range []string{
		"OBSERVE_USER_EMAIL",
		"OBSERVE_USER_PASSWORD",
		"OBSERVE_OAUTH2_CLIENT_ID",
		"OBSERVE_OAUTH2_CLIENT_SECRET",
		"OBSERVE_OAUTH2_OIDC_TOKEN",
		"OBSERVE_OAUTH2_OIDC_TOKEN_FILE_PATH",
		"OBSERVE_OAUTH2_OIDC_AUDIENCE",
		"OBSERVE_OAUTH2_TOKEN_URL",
		"OBSERVE_OAUTH2_SCOPE",
	} {
		t.Setenv(name, "")
	}
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"observe": func() (*schema.Provider, error) {
				p := Provider()
				p.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
					return client, nil
				}
				return p, nil
			},
		},
		Steps: steps,
	})
}

func TestReferenceTableAllowRowEditsKeepsEditedRows(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	original := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	changed := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\nweb-2,dev\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(original, true),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableRowEditsConfig(original, true),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex("host,owner\nweb-1,ops\n")),
			),
		},
		{
			Config: referenceTableRowEditsConfig(changed, true),
			Check:  checkReferenceTableAPICalls(api, 2, 0),
		},
	})
}

func TestReferenceTableRowEditsReplacedByDefault(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(source, false),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableRowEditsConfig(source, false),
			Check:     checkReferenceTableAPICalls(api, 2, 0),
		},
	})
}

func TestReferenceTableEnablingAllowRowEditsCallsNoAPI(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(source, false),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			Config: referenceTableRowEditsConfig(source, true),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "allow_row_edits", "true"),
			),
		},
	})
}

func TestDataSourceReferenceTableReadAfterRowEdits(t *testing.T) {
	api := &fakeReferenceTableAPI{checksum: md5Hex("host,owner\nweb-1,ops\n")}
	api.editRows()
	client := newFakeReferenceTableClient(t, api)
	data := schema.TestResourceDataRaw(t, dataSourceReferenceTable().Schema, map[string]interface{}{"id": fakeReferenceTableID})
	if diags := dataSourceReferenceTableRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read reference table data source: %v", diags)
	}
	if got := data.Get("checksum"); got != "" {
		t.Errorf("checksum = %q, want the API's empty checksum", got)
	}
}

func TestReferenceTableFailedUploadIsRetried(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	original := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	changed := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\nweb-2,dev\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(original, true),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: func() {
				api.editRows()
				api.setRejectUpdates(true)
			},
			Config:      referenceTableRowEditsConfig(changed, true),
			ExpectError: regexp.MustCompile("failed to update reference table"),
		},
		{
			PreConfig: func() { api.setRejectUpdates(false) },
			Config:    referenceTableRowEditsConfig(changed, true),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex("host,owner\nweb-1,ops\nweb-2,dev\n")),
			),
		},
	})
}

func TestReferenceTableDisallowingRowEditsUploadsEditedTable(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(source, true),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableRowEditsConfig(source, false),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "allow_row_edits", "false"),
			),
		},
	})
}

func TestReferenceTableDisallowingRowEditsWithoutEditsCallsNoAPI(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(source, true),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			Config: referenceTableRowEditsConfig(source, false),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
	})
}

func TestReferenceTableImportVerifies(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableRowEditsConfig(source, false),
		},
		{
			ResourceName:            "observe_reference_table.hosts",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"source_file"},
		},
	})
}

func TestReferenceTableStateWithoutAllowRowEditsPlansNoChange(t *testing.T) {
	source := writeReferenceTableCSV(t, "hosts.csv", "host,owner\nweb-1,ops\n")
	checksum := md5Hex("host,owner\nweb-1,ops\n")
	client := newFakeReferenceTableClient(t, &fakeReferenceTableAPI{checksum: checksum})
	r := resourceReferenceTable()
	data := r.Data(&terraform.InstanceState{
		ID: fakeReferenceTableID,
		Attributes: map[string]string{
			"label":         "hosts",
			"source_file":   source,
			"checksum":      checksum,
			"primary_key.#": "1",
			"primary_key.0": "host",
			"label_field":   "owner",
		},
	})
	if diags := resourceReferenceTableRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read reference table: %v", diags)
	}
	config := terraform.NewResourceConfigRaw(map[string]interface{}{
		"label":       "hosts",
		"source_file": source,
		"checksum":    checksum,
		"primary_key": []interface{}{"host"},
		"label_field": "owner",
	})
	diff, err := r.Diff(context.Background(), data.State(), config, client)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Empty() {
		t.Errorf("plan after refresh is not empty: %v", diff.Attributes)
	}
}

func md5Hex(contents string) string {
	sum := md5.Sum([]byte(contents))
	return hex.EncodeToString(sum[:])
}
