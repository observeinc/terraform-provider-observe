package binding

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDeserializeBindingsObject(t *testing.T) {
	jsonInput := `
	{
	  "kinds": [
		"dataset",
		"workspace"
	  ],
	  "mappings": {
		"dataset:Observe Dashboard": {
			"tf_local_binding_var": "binding__dashboard_bindings_test_dashboard__dataset_observe_dashboard",
			"tf_name":              "observe_dashboard"
		},
		"dataset:usage/Monitor Messages": {
			"tf_local_binding_var": "binding__dashboard_bindings_test_dashboard__dataset_monitor_messages",
			"tf_name":              "monitor_messages"
		}
	  },
	  "workspace": {
		"tf_local_binding_var": "binding__dashboard_bindings_test_dashboard__workspace_default",
		"tf_name":              "default"
	  },
	  "workspace_name": "default"
	}
	`
	var bindingsObj BindingsObject
	err := json.Unmarshal([]byte(jsonInput), &bindingsObj)
	if err != nil {
		t.Fatal(err)
	}
	expectedKinds := []Kind{KindDataset, KindWorkspace}
	if !reflect.DeepEqual(expectedKinds, bindingsObj.Kinds) {
		t.Fatalf("Expected %#v, got %#v", expectedKinds, bindingsObj.Kinds)
	}
	expectedMappings := Mapping{
		Ref{Kind: KindDataset, Key: "Observe Dashboard"}: Target{
			TfLocalBindingVar: "binding__dashboard_bindings_test_dashboard__dataset_observe_dashboard",
			TfName:            "observe_dashboard",
		},
		Ref{Kind: KindDataset, Key: "usage/Monitor Messages"}: Target{
			TfLocalBindingVar: "binding__dashboard_bindings_test_dashboard__dataset_monitor_messages",
			TfName:            "monitor_messages",
		},
	}
	if !reflect.DeepEqual(expectedMappings, bindingsObj.Mappings) {
		t.Fatalf("Expected %#v, got %#v", expectedMappings, bindingsObj.Mappings)
	}
	expectedWorkspace := Target{
		TfLocalBindingVar: "binding__dashboard_bindings_test_dashboard__workspace_default",
		TfName:            "default",
	}
	if !reflect.DeepEqual(expectedWorkspace, bindingsObj.Workspace) {
		t.Fatalf("Expected %#v, got %#v", expectedWorkspace, bindingsObj.Workspace)
	}
	expectedName := "default"
	if bindingsObj.WorkspaceName != expectedName {
		t.Fatalf("Expected workspace_name %s, got %s", expectedName, bindingsObj.WorkspaceName)
	}
}
