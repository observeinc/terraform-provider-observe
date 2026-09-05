package binding

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	inputJson = `
	{
	  "bv": false,
	  "datasetId": "41000123",
	  "id": "41000123",
	  "iv": 1231231,
	  "nested_field": {
		"dataset": "41000123",
		"datasetId": "1231231",
		"id": "41000201",
		"sv": "1231231",
		"targetDataset": "41000200"
	  },
	  "sv": "41000121",
	  "userId": "41000100",
	  "workspaceId": "o:::workspace:41000001"
	}
	`
	expectedJson = `
	{
	  "bv": false,
	  "datasetId": "${local.binding__type_name__dataset_dataset_1}",
	  "id": "${local.binding__type_name__dataset_dataset_1}",
	  "iv": 1231231,
	  "nested_field": {
		"dataset": "${local.binding__type_name__dataset_dataset_1}",
		"datasetId": "1231231",
		"id": "${local.binding__type_name__worksheet_worksheet_1}",
		"sv": "1231231",
		"targetDataset": "${local.binding__type_name__dataset_dataset_2}"
	  },
	  "sv": "41000121",
	  "userId": "${local.binding__type_name__user_basic_user}",
	  "workspaceId": "${local.binding__type_name__workspace_test_wks}"
	}
	`
	dataset1Id = "41000123"
)

func prepareResourceCacheFixture() ResourceCache {
	workspaceId := "41000001"
	r := ResourceCache{
		idToLabel:       make(map[Ref]ResourceCacheEntry),
		workspaceOid:    &oid.OID{Type: oid.TypeWorkspace, Id: workspaceId},
		forResourceKind: "type",
		forResourceName: "name",
	}
	disambiguator := 1
	existingResourceNames := make(map[string]struct{})
	r.addEntry(KindDataset, "dataset_1", "dataset_1", dataset1Id, true, &disambiguator, existingResourceNames)
	r.addEntry(KindDataset, "dataset_2", "dataset_2", "41000200", true, &disambiguator, existingResourceNames)
	r.addEntry(KindWorkspace, "Test wks", "Test wks", workspaceId, false, &disambiguator, existingResourceNames)
	r.addEntry(KindWorksheet, "worksheet_1", "worksheet_1", "41000201", true, &disambiguator, existingResourceNames)
	r.addEntry(KindUser, "basic@example.com", "basic_user", "41000100", true, &disambiguator, existingResourceNames)
	r.workspaceEntry = r.LookupId(KindWorkspace, workspaceId)
	return r
}

func prepareGeneratorFixture() Generator {
	return Generator{
		resourceName:    "name",
		resourceType:    "type",
		enabledBindings: NewKindSet(KindWorksheet, KindDataset, KindWorkspace, KindUser),
		bindings:        NewMapping(),
		cache:           prepareResourceCacheFixture(),
	}
}

func TestTryBindId(t *testing.T) {
	g := prepareGeneratorFixture()
	binding, _ := g.TryBindId(KindDataset, "41000123")
	expectedBinding := "${local.binding__type_name__dataset_dataset_1}"
	if binding != expectedBinding {
		t.Fatalf("expected binding %s, got actual binding %s", expectedBinding, binding)
	}
	binding, _ = g.TryBindId(KindDataset, "not_a_dataset_id")
	expectedBinding = "not_a_dataset_id"
	if binding != expectedBinding {
		t.Fatalf("Expected no binding '%s', got binding %s", expectedBinding, binding)
	}
}

func TestGenerate(t *testing.T) {
	var input map[string]interface{}
	var expected map[string]interface{}
	if err := json.Unmarshal([]byte(inputJson), &input); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(expectedJson), &expected); err != nil {
		t.Fatal(err)
	}
	g := prepareGeneratorFixture()
	g.Generate(input)
	if !reflect.DeepEqual(input, expected) {
		t.Fatalf("expected %#v, got %#v", expected, input)
	}
}

func TestGenerateJson(t *testing.T) {
	g := prepareGeneratorFixture()
	outputJson, err := g.GenerateJson([]byte(inputJson))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]interface{}
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(expectedJson), &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(outputJson, &output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output, expected) {
		t.Fatalf("expected %#v, got %#v", expected, output)
	}
}

func TestGenerateWithArrays(t *testing.T) {
	g := prepareGeneratorFixture()

	// scalars inside an array should be walked and replaced with local refs
	input := map[string]interface{}{
		"users": []interface{}{
			// full OID for a user — should be bound via TryBindOid
			"o:::user:41000100",
		},
	}
	g.Generate(input)
	users, ok := input["users"].([]interface{})
	if !ok || len(users) != 1 {
		t.Fatalf("expected users to remain a slice of length 1, got %#v", input["users"])
	}
	expectedRef := "${local.binding__type_name__user_basic_user}"
	if users[0] != expectedRef {
		t.Fatalf("expected %s, got %s", expectedRef, users[0])
	}

	// arrays of maps should also be walked recursively
	g2 := prepareGeneratorFixture()
	input2 := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"datasetId": dataset1Id,
			},
		},
	}
	g2.Generate(input2)
	items, ok := input2["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected items slice length 1, got %#v", input2["items"])
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected items[0] to be a map, got %T", items[0])
	}
	expectedDatasetRef := "${local.binding__type_name__dataset_dataset_1}"
	if item["datasetId"] != expectedDatasetRef {
		t.Fatalf("expected %s, got %s", expectedDatasetRef, item["datasetId"])
	}

	// nested arrays of scalars (array of array of OID strings)
	g3 := prepareGeneratorFixture()
	inner := []interface{}{"o:::user:41000100"}
	input3 := map[string]interface{}{
		"nested": []interface{}{inner},
	}
	g3.Generate(input3)
	outerSlice, ok := input3["nested"].([]interface{})
	if !ok || len(outerSlice) != 1 {
		t.Fatalf("expected nested slice length 1, got %#v", input3["nested"])
	}
	innerSlice, ok := outerSlice[0].([]interface{})
	if !ok || len(innerSlice) != 1 {
		t.Fatalf("expected inner slice length 1, got %#v", outerSlice[0])
	}
	if innerSlice[0] != expectedRef {
		t.Fatalf("expected %s, got %s", expectedRef, innerSlice[0])
	}
}

// TestSanitizeIdentifierStableUnderEscaping verifies that escaping HCL template
// markers in a name does not shift the sanitized identifier. This is critical
// because sanitizeIdentifier determines resource addresses, import commands, and
// local-variable names — a change here would silently rename resources across
// all exported configs. The stability relies on replaceInvalid using a `+`
// quantifier (runs of invalid characters collapse to one underscore), so both
// `${` and `$${` collapse identically.
func TestSanitizeIdentifierStableUnderEscaping(t *testing.T) {
	tests := []struct {
		name     string
		raw      string // name before escaping
		escaped  string // name after escaping
		expected string // expected sanitized result (same for both)
	}{
		{
			name:     "interpolation marker",
			raw:      "a ${b} c",
			escaped:  "a $${b} c",
			expected: "a_b_c",
		},
		{
			name:     "directive marker",
			raw:      "a %{if x}y%{endif} z",
			escaped:  "a %%{if x}y%%{endif} z",
			expected: "a_if_x_y_endif_z",
		},
		{
			name:     "env-style name",
			raw:      "my ${env}-monitor",
			escaped:  "my $${env}-monitor",
			expected: "my_env_-monitor",
		},
		{
			name:     "already-escaped",
			raw:      "a $${b} c",
			escaped:  "a $$${b} c",
			expected: "a_b_c",
		},
		{
			name:     "no markers",
			raw:      "plain-name_123",
			escaped:  "plain-name_123",
			expected: "plain-name_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRaw := sanitizeIdentifier(tt.raw)
			gotEscaped := sanitizeIdentifier(tt.escaped)
			if gotRaw != gotEscaped {
				t.Errorf("sanitizeIdentifier diverges:\n  raw(%q)     = %q\n  escaped(%q) = %q",
					tt.raw, gotRaw, tt.escaped, gotEscaped)
			}
			if gotRaw != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.raw, gotRaw, tt.expected)
			}
		})
	}
}

func TestInsertBindingsObjectJson(t *testing.T) {
	g := prepareGeneratorFixture()
	g.TryBindId(KindDataset, dataset1Id)
	// g.bindings[Ref{kind: KindDataset, key: "dataset_1"}] = Target{
	// 	TfLocalBindingVar: g.fmtTfLocalVar(KindDataset, &, false),
	// 	TfName:            "dataset_1",
	// }
	g.enabledBindings = NewKindSet(KindDataset, KindWorkspace)
	jsonData := `
	{
	  "data_fld_1": "value"
	}
	`
	expected := map[string]interface{}{
		"data_fld_1": "value",
		"bindings": map[string]interface{}{
			"mappings": map[string]interface{}{
				"dataset:dataset_1": map[string]interface{}{
					"tf_local_binding_var": "binding__type_name__dataset_dataset_1",
					"tf_name":              "type_name__dataset_dataset_1",
					"is_oid":               false,
				},
			},
			"kinds": []interface{}{
				"dataset",
				"workspace",
			},
			"workspace": map[string]interface{}{
				"tf_local_binding_var": "binding__type_name__workspace_test_wks",
				"tf_name":              "workspace_test_wks",
				"is_oid":               true,
			},
			"workspace_name": "Test wks",
		},
	}
	outputJson, err := g.InsertBindingsObjectJson([]byte(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]interface{}
	err = json.Unmarshal(outputJson, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output, expected) {
		t.Fatalf("expected %#v, got %#v", expected, output)
	}
}
