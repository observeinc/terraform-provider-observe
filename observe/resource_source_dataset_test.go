package observe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

const testStorageIntegrationOID = "o:::storageintegration:41009999"

// sourceDatasetCtyObject builds a cty object of the resource's implied type
// from simple Go values; attributes not in values are null.
func sourceDatasetCtyObject(t *testing.T, values map[string]interface{}) cty.Value {
	t.Helper()
	ty := resourceSourceDataset().CoreConfigSchema().ImpliedType()
	attrs := map[string]cty.Value{}
	for name, attrTy := range ty.AttributeTypes() {
		v, ok := values[name]
		if !ok {
			attrs[name] = cty.NullVal(attrTy)
			continue
		}
		switch x := v.(type) {
		case string:
			attrs[name] = cty.StringVal(x)
		case bool:
			attrs[name] = cty.BoolVal(x)
		case int:
			attrs[name] = cty.NumberIntVal(int64(x))
		case map[string]interface{}:
			m := map[string]cty.Value{}
			for k, mv := range x {
				m[k] = cty.StringVal(mv.(string))
			}
			if len(m) == 0 {
				attrs[name] = cty.MapValEmpty(cty.String)
			} else {
				attrs[name] = cty.MapVal(m)
			}
		default:
			t.Fatalf("unsupported value %T for %s", v, name)
		}
	}
	return cty.ObjectVal(attrs)
}

func flatStateAttributes(values map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range values {
		switch x := v.(type) {
		case string:
			out[k] = x
		case bool:
			if x {
				out[k] = "true"
			} else {
				out[k] = "false"
			}
		case int:
			out[k] = itoa(x)
		case map[string]interface{}:
			out[k+".%"] = itoa(len(x))
			for mk, mv := range x {
				out[k+"."+mk] = mv.(string)
			}
		}
	}
	return out
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// planSourceDataset plans config against an optional prior state (nil means create).
func planSourceDataset(t *testing.T, prior map[string]interface{}, config map[string]interface{}) (*terraform.InstanceDiff, error) {
	t.Helper()
	r := resourceSourceDataset()
	state := &terraform.InstanceState{RawConfig: sourceDatasetCtyObject(t, config)}
	if prior != nil {
		state.ID = "41000001"
		state.Attributes = flatStateAttributes(prior)
		state.Attributes["id"] = state.ID
		state.RawState = sourceDatasetCtyObject(t, prior)
	}
	return r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
}

func TestSourceDatasetResourceSchema(t *testing.T) {
	r := resourceSourceDataset()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("invalid resource schema: %s", err)
	}
	if r.SchemaVersion != 1 || len(r.StateUpgraders) != 1 || r.StateUpgraders[0].Version != 0 {
		t.Fatalf("unexpected schema version %d / upgraders %#v", r.SchemaVersion, r.StateUpgraders)
	}
	for _, k := range []string{"schema_type", "storage", "storage_integration", "external_write"} {
		if !r.Schema[k].ForceNew || r.Schema[k].Computed {
			t.Errorf("%s must be ForceNew and not Computed, so an omitted backing plans a replace", k)
		}
	}
	if r.Schema["deletion_protection"].Default != true {
		t.Errorf("deletion_protection must default to true")
	}
}

func TestSourceDatasetLegacyStateUpgrade(t *testing.T) {
	up := resourceSourceDataset().StateUpgraders[0]
	_, err := up.Upgrade(context.Background(), map[string]interface{}{
		"id": "41000001", "name": "x", "table_name": "T", "workspace": "o:::workspace:1",
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "terraform state rm") {
		t.Fatalf("expected legacy state error, got %v", err)
	}
}

// applySourceDatasetUpdate plans config against prior state and applies the
// plan through the SDK, returning the PATCH body the update would send.
func applySourceDatasetUpdate(t *testing.T, prior, config map[string]interface{}) string {
	t.Helper()
	diff, err := planSourceDataset(t, prior, config)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil || diff.RequiresNew() {
		t.Fatalf("expected an in-place update, got %#v", diff)
	}
	var body []byte
	r := resourceSourceDataset()
	r.UpdateContext = func(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
		b, err := json.Marshal(sourceDatasetUpdateRequest(data))
		if err != nil {
			return diag.FromErr(err)
		}
		body = b
		return nil
	}
	state := &terraform.InstanceState{
		ID:         "41000001",
		Attributes: flatStateAttributes(prior),
		RawState:   sourceDatasetCtyObject(t, prior),
		RawConfig:  sourceDatasetCtyObject(t, config),
	}
	state.Attributes["id"] = state.ID
	if _, diags := r.Apply(context.Background(), state, diff, nil); diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	return string(body)
}

func TestSourceDatasetRetentionZeroFromInherited(t *testing.T) {
	// Read stores the API's dataRetentionDays: null as -1.
	prior := map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Native", "schema_type": "Any", "external_write": false,
		"force_destroy": false, "deletion_protection": true, "data_retention_days": -1,
	}
	config := map[string]interface{}{"label": "tf-pkg/Logs", "data_retention_days": 0}
	if got, want := applySourceDatasetUpdate(t, prior, config), `{"dataRetentionDays":0}`; got != want {
		t.Fatalf("PATCH body = %s, want %s", got, want)
	}
}

func TestValidateSourceDatasetPlan(t *testing.T) {
	existing := map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Native", "schema_type": "Any", "external_write": false,
		"force_destroy": false,
	}
	cases := []struct {
		name    string
		prior   map[string]interface{}
		config  map[string]interface{}
		wantErr string
	}{
		{"native default", nil, map[string]interface{}{"label": "tf-pkg/a"}, ""},
		{"iceberg without integration", nil, map[string]interface{}{"label": "tf-pkg/a", "storage": "Iceberg"}, "requires storage_integration"},
		{"integration with native", nil, map[string]interface{}{"label": "tf-pkg/a", "storage": "Native", "storage_integration": testStorageIntegrationOID}, "requires storage = \"Iceberg\""},
		{"integration with default storage", nil, map[string]interface{}{"label": "tf-pkg/a", "storage_integration": testStorageIntegrationOID}, "requires storage = \"Iceberg\""},
		{"external write with native", nil, map[string]interface{}{"label": "tf-pkg/a", "external_write": true}, "external_write = true requires"},
		{"iceberg external write", nil, map[string]interface{}{"label": "tf-pkg/a", "storage": "Iceberg", "storage_integration": testStorageIntegrationOID, "external_write": true}, ""},
		{"package change", existing, map[string]interface{}{"label": "tf-other/Logs"}, "package of a source dataset label cannot change"},
		{"name change", existing, map[string]interface{}{"label": "tf-pkg/Renamed"}, ""},
		{"native to iceberg needs integration", existing, map[string]interface{}{"label": "tf-pkg/Logs", "storage": "Iceberg"}, "requires storage_integration"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := planSourceDataset(t, tc.prior, tc.config)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// An existing dataset's backing must be in configuration: the backing
// attributes default to Native, Any and no external write, so leaving one out
// plans a replacement, which deletion_protection fails at plan time.
func TestSourceDatasetBackingDefaultsPlanReplace(t *testing.T) {
	existingTypedIceberg := func(deletionProtection, forceDestroy bool) map[string]interface{} {
		return map[string]interface{}{
			"label": "tf-pkg/Logs", "storage": "Iceberg", "schema_type": "OtelLogs", "external_write": true,
			"storage_integration": testStorageIntegrationOID, "force_destroy": forceDestroy, "deletion_protection": deletionProtection,
		}
	}
	backing := map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Iceberg", "schema_type": "OtelLogs", "external_write": true,
		"storage_integration": testStorageIntegrationOID,
	}
	without := func(keys ...string) map[string]interface{} {
		out := map[string]interface{}{}
		for k, v := range backing {
			out[k] = v
		}
		for _, k := range keys {
			delete(out, k)
		}
		return out
	}
	with := func(config map[string]interface{}, key string, value interface{}) map[string]interface{} {
		out := map[string]interface{}{key: value}
		for k, v := range config {
			out[k] = v
		}
		return out
	}
	allBacking := []string{"schema_type", "storage", "storage_integration", "external_write"}
	cases := []struct {
		name        string
		prior       map[string]interface{}
		config      map[string]interface{}
		wantReplace []string
		wantErr     string
	}{
		{"full backing in config", existingTypedIceberg(true, false), backing, nil, ""},
		{"protected: schema_type omitted", existingTypedIceberg(true, false), without("schema_type"), nil,
			`this plan replaces source dataset 41000001, which deletes its data, because of schema_type "OtelLogs" -> "Any"; deletion_protection is true`},
		{"protected: external_write omitted", existingTypedIceberg(true, false), without("external_write"), nil, `external_write "true" -> "false"`},
		{"protected: backing omitted", existingTypedIceberg(true, false), map[string]interface{}{"label": "tf-pkg/Logs"}, nil,
			`schema_type "OtelLogs" -> "Any", storage "Iceberg" -> "Native", storage_integration "` + testStorageIntegrationOID + `" -> "", external_write "true" -> "false"`},
		// Delete reads prior state, so turning protection off in the same plan
		// must not let the replacement through.
		{"protected: disabled in the same plan", existingTypedIceberg(true, false), with(without("schema_type"), "deletion_protection", false), nil, "deletion_protection is true"},
		{"unprotected: backing omitted", existingTypedIceberg(false, false), map[string]interface{}{"label": "tf-pkg/Logs", "deletion_protection": false}, allBacking, ""},
		{"force_destroy overrides", existingTypedIceberg(true, true), map[string]interface{}{"label": "tf-pkg/Logs", "force_destroy": true}, allBacking, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := planSourceDataset(t, tc.prior, tc.config)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.wantReplace) == 0 {
				if diff != nil && len(diff.Attributes) > 0 {
					t.Fatalf("expected no changes, got %#v", diff.Attributes)
				}
				return
			}
			if diff == nil || !diff.RequiresNew() {
				t.Fatalf("expected a replacement, got %#v", diff)
			}
			for _, k := range tc.wantReplace {
				if a := diff.Attributes[k]; a == nil || !a.RequiresNew {
					t.Errorf("%s: expected a change that forces replacement, got %#v", k, a)
				}
			}
		})
	}
}

func newSourceDatasetTestClient(t *testing.T, handler http.HandlerFunc) *observeclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &observeclient.Client{Config: &observeclient.Config{}, Rest: rest.New(server.URL, server.Client())}
}

const testSourceDatasetResponse = `{"id":"41000001","label":"tf-pkg/a","description":"written","objectTags":{},` +
	`"dataRetentionDays":30,"effectiveDataRetentionDays":30,"dataSource":{"kind":"Ingest","ingest":{"storage":"Native","schemaType":"Any"}}}`

func TestSourceDatasetDeletionProtection(t *testing.T) {
	r := resourceSourceDataset()
	cases := []struct {
		deletionProtection, forceDestroy bool
		wantBlocked                      bool
	}{
		{true, false, true},
		{true, true, false},
		{false, false, false},
		{false, true, false},
	}
	for _, tc := range cases {
		d := r.Data(&terraform.InstanceState{ID: "41000001", Attributes: map[string]string{
			"id": "41000001", "deletion_protection": boolString(tc.deletionProtection), "force_destroy": boolString(tc.forceDestroy),
		}})
		err := checkSourceDatasetDeletionAllowed(d, d.Id())
		if blocked := err != nil; blocked != tc.wantBlocked {
			t.Errorf("deletion_protection=%t force_destroy=%t: blocked=%t (%v), want %t", tc.deletionProtection, tc.forceDestroy, blocked, err, tc.wantBlocked)
		}
	}
	protected := func() *schema.ResourceData {
		return r.Data(&terraform.InstanceState{ID: "41000001", Attributes: map[string]string{"id": "41000001", "deletion_protection": "true"}})
	}
	t.Run("existing dataset is kept", func(t *testing.T) {
		client := newSourceDatasetTestClient(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet {
				t.Errorf("a protected delete must not call %s", req.Method)
			}
			_, _ = w.Write([]byte(testSourceDatasetResponse))
		})
		d := protected()
		diags := resourceSourceDatasetDelete(context.Background(), d, client)
		if !diags.HasError() || !strings.Contains(diags[0].Summary, "terraform untaint") {
			t.Fatalf("expected deletion_protection error naming terraform untaint, got %v", diags)
		}
		if d.Id() == "" {
			t.Fatal("a refused delete must keep the resource in state")
		}
	})
	t.Run("dataset already deleted outside Terraform is cleared", func(t *testing.T) {
		client := newSourceDatasetTestClient(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet {
				t.Errorf("a protected delete must not call %s", req.Method)
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"type":"not_found","message":"source dataset not found"}`))
		})
		d := protected()
		if diags := resourceSourceDatasetDelete(context.Background(), d, client); diags.HasError() {
			t.Fatalf("unexpected error: %v", diags)
		}
		if d.Id() != "" {
			t.Fatal("expected the deleted dataset to be removed from state")
		}
	})
}

// Create and update take state from the API response: a follow-up GET can
// read the backend's caches before they refresh.
func TestSourceDatasetWriteUsesResponse(t *testing.T) {
	var methods []string
	client := newSourceDatasetTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		if req.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if req.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(testSourceDatasetResponse))
	})
	r := resourceSourceDataset()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30})
	if diags := resourceSourceDatasetCreate(context.Background(), d, client); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if d.Id() != "41000001" || d.Get("description") != "written" || d.Get("data_retention_days") != 30 {
		t.Fatalf("state not taken from the create response: id=%q description=%q retention=%v", d.Id(), d.Get("description"), d.Get("data_retention_days"))
	}

	prior := map[string]interface{}{
		"label": "tf-pkg/a", "storage": "Native", "schema_type": "Any", "external_write": false,
		"force_destroy": false, "deletion_protection": true, "data_retention_days": -1,
	}
	config := map[string]interface{}{"label": "tf-pkg/a", "description": "written", "data_retention_days": 30}
	diff, err := planSourceDataset(t, prior, config)
	if err != nil {
		t.Fatal(err)
	}
	state := &terraform.InstanceState{ID: "41000001", Attributes: flatStateAttributes(prior), RawState: sourceDatasetCtyObject(t, prior), RawConfig: sourceDatasetCtyObject(t, config)}
	state.Attributes["id"] = state.ID
	newState, diags := r.Apply(context.Background(), state, diff, client)
	if diags.HasError() {
		t.Fatalf("update: %v", diags)
	}
	if got := newState.Attributes["data_retention_days"]; got != "30" {
		t.Fatalf("state not taken from the update response: retention=%q", got)
	}
	if strings.Join(methods, ",") != "POST,PATCH" {
		t.Fatalf("requests = %v, want POST then PATCH with no GET", methods)
	}
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestSourceDatasetImportTypedNoReplace(t *testing.T) {
	prior := map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Iceberg", "schema_type": "OtelLogs", "external_write": false,
		"storage_integration": testStorageIntegrationOID, "force_destroy": false, "deletion_protection": true, "data_retention_days": 30,
	}
	diff, err := planSourceDataset(t, prior, map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Iceberg", "schema_type": "OtelLogs",
		// a versioned OID must not plan a data-deleting replacement
		"storage_integration": testStorageIntegrationOID + "/3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil && diff.RequiresNew() {
		t.Fatalf("unexpected replacement: %#v", diff.Attributes)
	}
	if diff != nil && len(diff.Attributes) > 0 {
		t.Fatalf("expected no changes, got %#v", diff.Attributes)
	}
}

func TestSourceDatasetCreateRequest(t *testing.T) {
	r := resourceSourceDataset()
	t.Run("explicit -1 sends no retention", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": -1})
		req, err := sourceDatasetCreateRequest(d)
		if err != nil {
			t.Fatal(err)
		}
		if req.DataRetentionDays != nil {
			t.Fatalf("-1 must not be sent, got %d", *req.DataRetentionDays)
		}
	})
	t.Run("defaults", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"label": "tf-pkg/a"})
		req, err := sourceDatasetCreateRequest(d)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(req)
		want := `{"label":"tf-pkg/a","dataSource":{"kind":"Ingest","ingest":{"storage":"Native","schemaType":"Any"}}}`
		if string(body) != want {
			t.Fatalf("body = %s, want %s", body, want)
		}
	})
	t.Run("iceberg with retention zero and tags", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
			"label": "tf-pkg/a", "storage": "Iceberg", "storage_integration": testStorageIntegrationOID,
			"external_write": true, "data_retention_days": 0, "object_tags": map[string]interface{}{"team": "a,b"},
		})
		req, err := sourceDatasetCreateRequest(d)
		if err != nil {
			t.Fatal(err)
		}
		if req.DataRetentionDays == nil || *req.DataRetentionDays != 0 {
			t.Errorf("explicit 0 retention must be sent, got %v", req.DataRetentionDays)
		}
		ice := req.DataSource.Ingest.Iceberg
		if ice == nil || ice.StorageIntegrationId != "41009999" || !ice.ExternalWrite {
			t.Errorf("unexpected iceberg %#v", ice)
		}
		if len(req.ObjectTags["team"]) != 2 {
			t.Errorf("unexpected tags %#v", req.ObjectTags)
		}
	})
}

func TestSourceDatasetUpdateRequest(t *testing.T) {
	prior := map[string]interface{}{
		"label": "tf-pkg/a", "description": "old", "data_retention_days": 30, "object_tags": map[string]interface{}{"team": "a"},
		"storage": "Native", "schema_type": "Any", "external_write": false, "force_destroy": false, "deletion_protection": true,
	}
	cases := []struct {
		name   string
		config map[string]interface{}
		want   string
	}{
		{
			// No dataRetentionDays key, so a non-admin can edit metadata.
			name:   "clear description and remove tags",
			config: map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30},
			want:   `{"description":null,"objectTags":{}}`,
		},
		{
			name:   "rename and set retention to zero",
			config: map[string]interface{}{"label": "tf-pkg/b", "description": "old", "data_retention_days": 0, "object_tags": map[string]interface{}{"team": "a"}},
			want:   `{"dataRetentionDays":0,"label":"tf-pkg/b"}`,
		},
		{
			name:   "removing retention from config sends nothing for it",
			config: map[string]interface{}{"label": "tf-pkg/b", "description": "old", "object_tags": map[string]interface{}{"team": "a"}},
			want:   `{"label":"tf-pkg/b"}`,
		},
		{
			name:   "explicit -1 returns to inherited retention",
			config: map[string]interface{}{"label": "tf-pkg/a", "description": "old", "data_retention_days": -1, "object_tags": map[string]interface{}{"team": "a"}},
			want:   `{"dataRetentionDays":null}`,
		},
		{
			name:   "deletion_protection only",
			config: map[string]interface{}{"label": "tf-pkg/a", "description": "old", "data_retention_days": 30, "object_tags": map[string]interface{}{"team": "a"}, "deletion_protection": false},
			want:   `{}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := applySourceDatasetUpdate(t, prior, tc.config); got != tc.want {
				t.Fatalf("PATCH body = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestSourceDatasetReadNullRetention(t *testing.T) {
	r := resourceSourceDataset()
	d := r.Data(&terraform.InstanceState{ID: "41000001", Attributes: map[string]string{"id": "41000001", "data_retention_days": "30"}})
	ds := &rest.SourceDatasetResource{Id: "41000001", Label: "tf-pkg/a", EffectiveDataRetentionDays: 90}
	if diags := sourceDatasetToResourceData(d, ds); diags.HasError() {
		t.Fatal(diags)
	}
	if got := d.Get("data_retention_days").(int); got != sourceDatasetRetentionInherit {
		t.Fatalf("a null retention from the API must be stored as -1, got %d", got)
	}
	// A retention reset outside Terraform shows as drift against a configured value.
	prior := map[string]interface{}{
		"label": "tf-pkg/a", "storage": "Native", "schema_type": "Any", "external_write": false,
		"force_destroy": false, "deletion_protection": true, "data_retention_days": -1,
	}
	if got, want := applySourceDatasetUpdate(t, prior, map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30}), `{"dataRetentionDays":30}`; got != want {
		t.Fatalf("PATCH body = %s, want %s", got, want)
	}
}

func TestParseSourceDatasetImportID(t *testing.T) {
	for in, want := range map[string]string{"41000001": "41000001", "o:::dataset:41000001": "41000001", "o:::dataset:41000001/2": "41000001"} {
		got, err := parseSourceDatasetImportID(in)
		if err != nil || got != want {
			t.Errorf("parse(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "o:::workspace:1"} {
		if _, err := parseSourceDatasetImportID(in); err == nil {
			t.Errorf("parse(%q) should fail", in)
		}
	}
}
