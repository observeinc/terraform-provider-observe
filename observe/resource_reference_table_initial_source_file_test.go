package observe

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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
// MD5 of each uploaded file as the checksum and clears it when rows are edited,
// and its PATCH updates label, description and primaryKey but not labelField.
type fakeReferenceTableAPI struct {
	mu               sync.Mutex
	table            rest.ReferenceTable
	schema           []rest.ReferenceTableSchemaInput
	uploads          int
	metadataPatches  int
	deletes          int
	unexpectedRoutes []string
}

func (f *fakeReferenceTableAPI) editRows() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.table.Checksum = ""
}

func (f *fakeReferenceTableAPI) counts() (uploads, metadataPatches int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uploads, f.metadataPatches
}

func (f *fakeReferenceTableAPI) deleteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deletes
}

// datasetResponse returns the dataset typedef: the uploaded schema, or string
// host and owner columns when the upload had none.
func (f *fakeReferenceTableAPI) datasetResponse() ([]byte, error) {
	type fieldType struct {
		Rep string `json:"rep"`
	}
	type field struct {
		Name string    `json:"name"`
		Type fieldType `json:"type"`
	}
	fields := []field{{"host", fieldType{"string"}}, {"owner", fieldType{"string"}}}
	if len(f.schema) > 0 {
		fields = fields[:0]
		for _, column := range f.schema {
			fields = append(fields, field{column.Name, fieldType{column.Type}})
		}
	}
	response := map[string]interface{}{"data": map[string]interface{}{"dataset": map[string]interface{}{
		"id":      "41000002",
		"name":    "Reference Tables/hosts",
		"typedef": map[string]interface{}{"def": map[string]interface{}{"fields": fields}},
	}}}
	return json.Marshal(response)
}

// replaceTable applies a POST or PUT, which replaces the rows and all metadata.
func (f *fakeReferenceTableAPI) replaceTable(r *http.Request) error {
	file, _, err := r.FormFile("upload")
	if err != nil {
		return err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	var metadata rest.ReferenceTableMetadataInput
	if err := json.Unmarshal([]byte(r.FormValue("metadata")), &metadata); err != nil {
		return err
	}
	schemaFile, _, err := r.FormFile("schema")
	if err != nil {
		return err
	}
	defer schemaFile.Close()
	f.schema = nil
	if err := json.NewDecoder(schemaFile).Decode(&f.schema); err != nil {
		return err
	}
	f.table = rest.ReferenceTable{
		Id:         fakeReferenceTableID,
		DatasetId:  "41000002",
		Checksum:   hex.EncodeToString(hash.Sum(nil)),
		LabelField: metadata.LabelField,
	}
	if metadata.Label != nil {
		f.table.Label = *metadata.Label
	}
	if metadata.Description != nil {
		f.table.Description = *metadata.Description
	}
	if metadata.PrimaryKey != nil && len(*metadata.PrimaryKey) > 0 {
		f.table.PrimaryKey = *metadata.PrimaryKey
	}
	if f.table.LabelField != nil && *f.table.LabelField == "" {
		f.table.LabelField = nil
	}
	f.uploads++
	return nil
}

func (f *fakeReferenceTableAPI) patchMetadata(r *http.Request) error {
	var metadata rest.ReferenceTableMetadataInput
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		return err
	}
	if metadata.Label != nil {
		f.table.Label = *metadata.Label
	}
	if metadata.Description != nil {
		f.table.Description = *metadata.Description
	}
	if metadata.PrimaryKey != nil {
		f.table.PrimaryKey = *metadata.PrimaryKey
	}
	f.metadataPatches++
	return nil
}

func (f *fakeReferenceTableAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	tablePath := "/v1/referencetables/" + fakeReferenceTableID
	var err error
	switch {
	case r.URL.Path == "/v1/meta":
		response, err := f.datasetResponse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(response)
		return
	case r.URL.Path == "/v1/referencetables" && r.Method == http.MethodPost,
		r.URL.Path == tablePath && r.Method == http.MethodPut:
		err = f.replaceTable(r)
	case r.URL.Path == tablePath && r.Method == http.MethodPatch:
		err = f.patchMetadata(r)
	case r.URL.Path == tablePath && r.Method == http.MethodGet:
	case r.URL.Path == tablePath && r.Method == http.MethodDelete:
		f.deletes++
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		f.unexpectedRoutes = append(f.unexpectedRoutes, r.Method+" "+r.URL.Path)
		http.Error(w, "unexpected route", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(f.table)
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

// referenceTableConfig renders a reference table whose rows come from
// fileAttribute, either "source_file" or "initial_source_file". extra is
// appended inside the resource block.
func referenceTableConfig(fileAttribute, path, extra string) string {
	checksum := ""
	if fileAttribute == "source_file" {
		checksum = fmt.Sprintf("checksum = filemd5(%q)", path)
	}
	return fmt.Sprintf(`
resource "observe_reference_table" "hosts" {
  label       = "hosts"
  %s = %q
  %s
  primary_key = ["host"]
  label_field = "owner"
  %s
}
`, fileAttribute, path, checksum, extra)
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

const (
	originalHostsCSV = "host,owner\nweb-1,ops\n"
	changedHostsCSV  = "host,owner\nweb-1,ops\nweb-2,dev\n"
)

func TestReferenceTableInitialSourceFileKeepsEditedRows(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	original := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	changed := writeReferenceTableCSV(t, "hosts.csv", changedHostsCSV)
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableConfig("initial_source_file", original, ""),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex(originalHostsCSV)),
			),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableConfig("initial_source_file", original, ""),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", ""),
			),
		},
		{
			Config: referenceTableConfig("initial_source_file", changed, ""),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			Config: referenceTableConfig("initial_source_file", changed, `description = "owners by host"`),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 1),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "description", "owners by host"),
			),
		},
		{
			Config: `
resource "observe_reference_table" "hosts" {
  label               = "hosts"
  initial_source_file = "` + changed + `"
  primary_key         = ["host", "owner"]
  label_field         = "owner"
  description         = "owners by host"
}
`,
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 2),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "primary_key.#", "2"),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", ""),
			),
		},
	})
}

func checkReferenceTableDeletes(api *fakeReferenceTableAPI, wantDeletes int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if deletes := api.deleteCount(); deletes != wantDeletes {
			return fmt.Errorf("deletes = %d, want %d", deletes, wantDeletes)
		}
		return nil
	}
}

func TestReferenceTableInitialSourceFileRejectsUploadOnlyChanges(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	initialConfig := func(labelField string) string {
		return fmt.Sprintf(`
resource "observe_reference_table" "hosts" {
  label               = "hosts"
  initial_source_file = %q
  primary_key         = ["host"]
  label_field         = %q
}
`, source, labelField)
	}
	ownerAsInt := `
  schema {
    name = "host"
    type = "string"
  }
  schema {
    name = "owner"
    type = "int64"
  }`
	noReplacement := resource.ComposeTestCheckFunc(
		checkReferenceTableDeletes(api, 0),
		resource.TestCheckResourceAttr("observe_reference_table.hosts", "id", fakeReferenceTableID),
		resource.TestCheckResourceAttr("observe_reference_table.hosts", "dataset", "o:::dataset:41000002"),
	)
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: initialConfig("owner"),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig:   api.editRows,
			Config:      initialConfig("host"),
			ExpectError: regexp.MustCompile(`label_field cannot be changed on a reference table created from\s+initial_source_file`),
		},
		{
			Config:      referenceTableConfig("initial_source_file", source, ownerAsInt),
			ExpectError: regexp.MustCompile(`schema cannot be changed on a reference table created from\s+initial_source_file`),
		},
		{
			Config: initialConfig("owner"),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 1, 0),
				noReplacement,
			),
		},
		{
			Config: referenceTableConfig("source_file", source, ownerAsInt),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				noReplacement,
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "schema.1.type", "int64"),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex(originalHostsCSV)),
			),
		},
		{
			Config: referenceTableConfig("initial_source_file", source, ownerAsInt),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				noReplacement,
			),
		},
	})
}

func TestReferenceTableSourceFileSchemaChangeUploadsInPlace(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableConfig("source_file", source, ""),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			Config: referenceTableConfig("source_file", source, `
  schema {
    name = "host"
    type = "string"
  }
  schema {
    name = "owner"
    type = "int64"
  }`),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				checkReferenceTableDeletes(api, 0),
			),
		},
	})
}

func TestReferenceTableSourceFileReplacesEditedRows(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableConfig("source_file", source, ""),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableConfig("source_file", source, ""),
			Check: resource.ComposeTestCheckFunc(
				checkReferenceTableAPICalls(api, 2, 0),
				resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex(originalHostsCSV)),
			),
		},
	})
}

func TestReferenceTableSwitchingToInitialSourceFileCallsNoAPI(t *testing.T) {
	api := &fakeReferenceTableAPI{}
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
		{
			Config: referenceTableConfig("source_file", source, ""),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			Config: referenceTableConfig("initial_source_file", source, ""),
			Check:  checkReferenceTableAPICalls(api, 1, 0),
		},
		{
			PreConfig: api.editRows,
			Config:    referenceTableConfig("initial_source_file", source, ""),
			Check:     checkReferenceTableAPICalls(api, 1, 0),
		},
	})
}

func TestReferenceTableSwitchingToSourceFileUploadsOnlyAfterRowEdits(t *testing.T) {
	t.Run("rows unchanged", func(t *testing.T) {
		api := &fakeReferenceTableAPI{}
		source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
		testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
			{
				Config: referenceTableConfig("initial_source_file", source, ""),
				Check:  checkReferenceTableAPICalls(api, 1, 0),
			},
			{
				Config: referenceTableConfig("source_file", source, ""),
				Check:  checkReferenceTableAPICalls(api, 1, 0),
			},
		})
	})
	t.Run("rows edited", func(t *testing.T) {
		api := &fakeReferenceTableAPI{}
		source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
		testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
			{
				Config: referenceTableConfig("initial_source_file", source, ""),
				Check:  checkReferenceTableAPICalls(api, 1, 0),
			},
			{
				PreConfig: api.editRows,
				Config:    referenceTableConfig("source_file", source, ""),
				Check: resource.ComposeTestCheckFunc(
					checkReferenceTableAPICalls(api, 2, 0),
					resource.TestCheckResourceAttr("observe_reference_table.hosts", "checksum", md5Hex(originalHostsCSV)),
				),
			},
		})
	})
}

func TestReferenceTableSourceFileValidation(t *testing.T) {
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	for _, tc := range []struct {
		name      string
		fileAttrs string
		wantError string
	}{
		{
			name:      "both files",
			fileAttrs: fmt.Sprintf("source_file = %[1]q\n  checksum = filemd5(%[1]q)\n  initial_source_file = %[1]q", source),
			wantError: `only one of\s+.initial_source_file,source_file.\s+can\s+be\s+specified`,
		},
		{
			name:      "no file",
			fileAttrs: "",
			wantError: `one of\s+.initial_source_file,source_file.\s+must\s+be\s+specified`,
		},
		{
			name:      "checksum with initial_source_file",
			fileAttrs: fmt.Sprintf("initial_source_file = %[1]q\n  checksum = filemd5(%[1]q)", source),
			wantError: `"checksum":\s+conflicts\s+with\s+initial_source_file`,
		},
		{
			name:      "source_file without checksum",
			fileAttrs: fmt.Sprintf("source_file = %q", source),
			wantError: `all of\s+.checksum,source_file.\s+must\s+be\s+specified`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testReferenceTableWithFakeAPI(t, &fakeReferenceTableAPI{}, []resource.TestStep{
				{
					Config: fmt.Sprintf(`
resource "observe_reference_table" "hosts" {
  label = "hosts"
  %s
}
`, tc.fileAttrs),
					ExpectError: regexp.MustCompile(tc.wantError),
				},
			})
		})
	}
}

func TestReferenceTableImportVerifies(t *testing.T) {
	for _, fileAttribute := range []string{"source_file", "initial_source_file"} {
		t.Run(fileAttribute, func(t *testing.T) {
			api := &fakeReferenceTableAPI{}
			source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
			testReferenceTableWithFakeAPI(t, api, []resource.TestStep{
				{
					Config: referenceTableConfig(fileAttribute, source, ""),
				},
				{
					ResourceName:            "observe_reference_table.hosts",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"source_file", "initial_source_file"},
				},
			})
		})
	}
}

func TestDataSourceReferenceTableReadAfterRowEdits(t *testing.T) {
	api := &fakeReferenceTableAPI{table: rest.ReferenceTable{
		Id:        fakeReferenceTableID,
		Label:     "hosts",
		DatasetId: "41000002",
		Checksum:  md5Hex(originalHostsCSV),
	}}
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

// State written before initial_source_file existed must refresh to an empty plan.
func TestReferenceTableStateFromPreviousVersionPlansNoChange(t *testing.T) {
	source := writeReferenceTableCSV(t, "hosts.csv", originalHostsCSV)
	checksum := md5Hex(originalHostsCSV)
	labelField := "owner"
	client := newFakeReferenceTableClient(t, &fakeReferenceTableAPI{table: rest.ReferenceTable{
		Id:         fakeReferenceTableID,
		Label:      "hosts",
		DatasetId:  "41000002",
		Checksum:   checksum,
		PrimaryKey: []string{"host"},
		LabelField: &labelField,
	}})
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
