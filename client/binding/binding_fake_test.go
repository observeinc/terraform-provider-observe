package binding

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/observeinc/terraform-provider-observe/client/binding/bindingtest"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var localRef = regexp.MustCompile(`\$\{local\.([A-Za-z0-9_-]+)\}`)

func fakeTenant(datasets ...bindingtest.Object) bindingtest.Tenant {
	return bindingtest.Tenant{
		Workspace:  bindingtest.Object{ID: "41000001", Label: "Default"},
		Datasets:   datasets,
		Worksheets: []bindingtest.Object{{ID: "41000201", Label: "My Worksheet"}, {ID: "41000500", Label: "Shared Id Worksheet"}},
		Users:      []bindingtest.User{{ID: 41000100, Email: "basic@example.com", Label: "Basic User"}},
	}
}

// export binds data for one object against fake and returns the rewritten
// document and its bindings.
func export(t *testing.T, fake *bindingtest.Server, kinds KindSet, data map[string]interface{}) (map[string]interface{}, BindingsObject) {
	t.Helper()
	gen, err := NewGenerator(context.Background(), "dashboard", "Obj", fake.Client(), kinds)
	if err != nil {
		t.Fatal(err)
	}
	gen.Generate(data)
	bindings, err := gen.GetBindings()
	if err != nil {
		t.Fatal(err)
	}
	return data, bindings
}

// danglingRefs returns local references in data that no binding defines.
func danglingRefs(t *testing.T, data interface{}, b BindingsObject) []string {
	t.Helper()
	defined := map[string]bool{b.Workspace.TfLocalBindingVar: true}
	for _, target := range b.Mappings {
		defined[target.TfLocalBindingVar] = true
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var dangling []string
	for _, m := range localRef.FindAllStringSubmatch(string(raw), -1) {
		if !defined[m[1]] {
			dangling = append(dangling, m[1])
		}
	}
	return dangling
}

func TestFakeDatasetBeforeWorksheetForRawId(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000500", Label: "Shared Id Dataset"}))
	out, _ := export(t, fake, NewKindSet(KindDataset, KindWorksheet, KindWorkspace), map[string]interface{}{
		"both":      map[string]interface{}{"id": "41000500"},
		"worksheet": map[string]interface{}{"id": "41000201"},
	})
	want := map[string]string{
		"both":      "${local.binding__dashboard_obj__dataset_shared_id_dataset}",
		"worksheet": "${local.binding__dashboard_obj__worksheet_my_worksheet}",
	}
	for k, ref := range want {
		if got := out[k].(map[string]interface{})["id"]; got != ref {
			t.Errorf("%s: got %v, want %s", k, got, ref)
		}
	}
}

func TestFakeDisabledKindsUntouched(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	in := map[string]interface{}{
		"datasetId": "41000123",
		"oid":       "o:::dataset:41000123",
		"userId":    "41000100",
		"id":        "41000201",
	}
	out, b := export(t, fake, NewKindSet(KindWorkspace), in)
	for k, v := range map[string]string{"datasetId": "41000123", "oid": "o:::dataset:41000123", "userId": "41000100", "id": "41000201"} {
		if out[k] != v {
			t.Errorf("%s: got %v, want %s", k, out[k], v)
		}
	}
	if len(b.Mappings) != 0 {
		t.Errorf("got mappings %v, want none", b.Mappings)
	}
	for _, op := range []string{bindingtest.OpListDatasetsIdNameOnly, bindingtest.OpRestDatasets} {
		if n := fake.Count(op); n != 0 {
			t.Errorf("%s: got %d requests, want 0", op, n)
		}
	}
}

// Different labels that sanitize to the same name are disambiguated in
// response order.
func TestFakeSanitizedCollisionFollowsResponseOrder(t *testing.T) {
	a := bindingtest.Object{ID: "41000801", Label: "Foo/Bar"}
	b := bindingtest.Object{ID: "41000802", Label: "foo bar"}
	for _, tc := range []struct {
		order []bindingtest.Object
		want  map[string]string
	}{
		{[]bindingtest.Object{a, b}, map[string]string{"41000801": "foo_bar", "41000802": "foo_bar_1"}},
		{[]bindingtest.Object{b, a}, map[string]string{"41000801": "foo_bar_1", "41000802": "foo_bar"}},
	} {
		fake := bindingtest.New(t, fakeTenant(tc.order...))
		out, bindings := export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{
			"a": map[string]interface{}{"datasetId": "41000801"},
			"b": map[string]interface{}{"datasetId": "41000802"},
		})
		for k, id := range map[string]string{"a": "41000801", "b": "41000802"} {
			want := "${local.binding__dashboard_obj__dataset_" + tc.want[id] + "}"
			if got := out[k].(map[string]interface{})["datasetId"]; got != want {
				t.Errorf("%s: got %v, want %s", id, got, want)
			}
		}
		if d := danglingRefs(t, out, bindings); len(d) != 0 {
			t.Errorf("dangling refs %v", d)
		}
	}
}

// Two datasets with the same label share one mapping key; one reference is
// left without a binding.
func TestFakeDuplicateLabelLeavesDanglingRef(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(
		bindingtest.Object{ID: "41000901", Label: "Dup"},
		bindingtest.Object{ID: "41000902", Label: "Dup"},
	))
	out, bindings := export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{
		"a": map[string]interface{}{"datasetId": "41000901"},
		"b": map[string]interface{}{"datasetId": "41000902"},
	})
	if n := len(danglingRefs(t, out, bindings)); n != 1 {
		t.Fatalf("got %d dangling refs, want 1: %v", n, out)
	}
}

// A dataset used as raw ID and OID gets one mapping whose IsOid is the last use.
func TestFakeRawIdAndOidShareMapping(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	for _, oidLast := range []bool{false, true} {
		gen, err := NewGenerator(context.Background(), "dashboard", "Obj", fake.Client(), NewKindSet(KindDataset, KindWorkspace))
		if err != nil {
			t.Fatal(err)
		}
		dsOid := oid.OID{Type: oid.TypeDataset, Id: "41000123"}
		var refs [2]string
		if oidLast {
			refs[0], _ = gen.TryBindId(KindDataset, "41000123")
			refs[1], _ = gen.TryBindOid(dsOid)
		} else {
			refs[0], _ = gen.TryBindOid(dsOid)
			refs[1], _ = gen.TryBindId(KindDataset, "41000123")
		}
		if refs[0] != refs[1] {
			t.Fatalf("got distinct refs %v", refs)
		}
		b, err := gen.GetBindings()
		if err != nil {
			t.Fatal(err)
		}
		if got := b.Mappings[Ref{Kind: KindDataset, Key: "Logs"}].IsOid; got != oidLast {
			t.Errorf("oidLast=%v: got IsOid %v", oidLast, got)
		}
	}
}

func TestFakeGeneratorFailures(t *testing.T) {
	for _, op := range []string{bindingtest.OpListWorkspaces, bindingtest.OpListDatasetsIdNameOnly} {
		for _, f := range []bindingtest.Failure{bindingtest.FailUnauthorized, bindingtest.FailMalformed, bindingtest.FailNetwork} {
			fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
			fake.Fail(op, f)
			if _, err := NewGenerator(context.Background(), "dashboard", "Obj", fake.Client(), NewKindSet(KindDataset, KindWorkspace)); err == nil {
				t.Errorf("%s/%d: expected error", op, f)
			}
		}
	}
}
