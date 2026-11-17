package observe

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		if !r.Schema[k].ForceNew {
			t.Errorf("%s must be ForceNew", k)
		}
		if r.Schema[k].Default != nil {
			t.Errorf("%s must not have a Default", k)
		}
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
	if up.Type.IsObjectType() == false || !up.Type.HasAttribute("table_name") {
		t.Fatalf("upgrader type must be the legacy schema, got %#v", up.Type)
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

func TestSourceDatasetImportTypedNoReplace(t *testing.T) {
	prior := map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Iceberg", "schema_type": "OtelLogs", "external_write": false,
		"storage_integration": testStorageIntegrationOID, "force_destroy": false, "data_retention_days": 30,
	}
	diff, err := planSourceDataset(t, prior, map[string]interface{}{
		"label": "tf-pkg/Logs", "storage": "Iceberg",
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

// fakeSourceDatasetReader serves planned values and a set of changed keys.
type fakeSourceDatasetReader struct {
	values  map[string]interface{}
	changed map[string]bool
	raw     cty.Value
}

func (f fakeSourceDatasetReader) Get(k string) interface{} {
	if v, ok := f.values[k]; ok {
		return v
	}
	return resourceSourceDataset().Schema[k].ZeroValue()
}
func (f fakeSourceDatasetReader) GetOk(k string) (interface{}, bool) {
	v, ok := f.values[k]
	if m, isMap := v.(map[string]interface{}); isMap && len(m) == 0 {
		return v, false
	}
	return v, ok && v != ""
}
func (f fakeSourceDatasetReader) GetOkExists(k string) (interface{}, bool) {
	v, ok := f.values[k]
	return v, ok
}
func (f fakeSourceDatasetReader) HasChange(k string) bool { return f.changed[k] }
func (f fakeSourceDatasetReader) GetRawConfig() cty.Value { return f.raw }

func TestSourceDatasetUpdateRequest(t *testing.T) {
	cases := []struct {
		name    string
		values  map[string]interface{}
		config  map[string]interface{}
		changed []string
		want    string
	}{
		{
			// No dataRetentionDays key, so a non-admin can edit metadata.
			name:    "clear description and remove tags",
			values:  map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30, "object_tags": map[string]interface{}{}},
			config:  map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30},
			changed: []string{"description", "object_tags"},
			want:    `{"description":null,"objectTags":{}}`,
		},
		{
			name:    "rename and set retention",
			values:  map[string]interface{}{"label": "tf-pkg/b", "data_retention_days": 0},
			config:  map[string]interface{}{"label": "tf-pkg/b", "data_retention_days": 0},
			changed: []string{"label", "data_retention_days"},
			want:    `{"dataRetentionDays":0,"label":"tf-pkg/b"}`,
		},
		{
			name:    "removing retention from config sends nothing",
			values:  map[string]interface{}{"label": "tf-pkg/a", "data_retention_days": 30},
			config:  map[string]interface{}{"label": "tf-pkg/a"},
			changed: []string{"data_retention_days"},
			want:    `{}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := map[string]bool{}
			for _, k := range tc.changed {
				changed[k] = true
			}
			r := fakeSourceDatasetReader{values: tc.values, changed: changed, raw: sourceDatasetCtyObject(t, tc.config)}
			body, err := json.Marshal(sourceDatasetUpdateRequest(r))
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != tc.want {
				t.Fatalf("body = %s, want %s", body, tc.want)
			}
		})
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
