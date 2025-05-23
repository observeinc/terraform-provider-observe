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
	r.addEntry(KindDataset, "dataset_1", dataset1Id, true, &disambiguator, existingResourceNames)
	r.addEntry(KindDataset, "dataset_2", "41000200", true, &disambiguator, existingResourceNames)
	r.addEntry(KindWorkspace, "Test wks", workspaceId, false, &disambiguator, existingResourceNames)
	r.addEntry(KindWorksheet, "worksheet_1", "41000201", true, &disambiguator, existingResourceNames)
	r.addEntry(KindUser, "basic_user", "41000100", true, &disambiguator, existingResourceNames)
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
