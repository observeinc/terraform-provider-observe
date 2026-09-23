package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/observeinc/terraform-provider-observe/client/binding/bindingtest"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func fakeTenant(datasets ...bindingtest.Object) bindingtest.Tenant {
	return bindingtest.Tenant{
		Workspace:  bindingtest.Object{ID: "41000001", Label: "Default"},
		Datasets:   datasets,
		Worksheets: []bindingtest.Object{{ID: "41000201", Label: "My Worksheet"}, {ID: "41000500", Label: "Shared Id Worksheet"}},
		Users:      []bindingtest.User{{ID: 41000100, Email: "basic@example.com", Label: "Basic User"}},
	}
}

func newFakeGenerator(t *testing.T, fake *bindingtest.Server, kinds KindSet) Generator {
	t.Helper()
	gen, err := NewGenerator(context.Background(), "dashboard", "Obj", fake.Client(), kinds)
	if err != nil {
		t.Fatal(err)
	}
	return gen
}

// exportErr collects, resolves and binds data for one object against fake.
func exportErr(t *testing.T, fake *bindingtest.Server, kinds KindSet, data map[string]interface{}) (map[string]interface{}, BindingsObject, error) {
	t.Helper()
	gen := newFakeGenerator(t, fake, kinds)
	gen.Collect(data)
	if err := gen.Resolve(context.Background()); err != nil {
		return nil, BindingsObject{}, err
	}
	gen.Generate(data)
	bindings, err := gen.GetBindings()
	return data, bindings, err
}

func export(t *testing.T, fake *bindingtest.Server, kinds KindSet, data map[string]interface{}) (map[string]interface{}, BindingsObject) {
	t.Helper()
	out, bindings, err := exportErr(t, fake, kinds, data)
	if err != nil {
		t.Fatal(err)
	}
	if undefined := undefinedLocals(t, out, bindings); len(undefined) != 0 {
		t.Fatalf("references without bindings: %v", undefined)
	}
	return out, bindings
}

func undefinedLocals(t *testing.T, data interface{}, b BindingsObject) []string {
	t.Helper()
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	undefined, err := bindingtest.UndefinedLocals(data, raw)
	if err != nil {
		t.Fatal(err)
	}
	return undefined
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

// Different labels that sanitize to the same name are numbered in id order.
func TestFakeSanitizedCollisionOrderedById(t *testing.T) {
	a := bindingtest.Object{ID: "41000801", Label: "Foo/Bar"}
	b := bindingtest.Object{ID: "41000802", Label: "foo bar"}
	c := bindingtest.Object{ID: "9", Label: "foo_bar_1"}
	for _, order := range [][]bindingtest.Object{{a, b, c}, {c, b, a}} {
		fake := bindingtest.New(t, fakeTenant(order...))
		out, _ := export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{
			"a": map[string]interface{}{"datasetId": "41000801"},
			"b": map[string]interface{}{"datasetId": "41000802"},
			"c": map[string]interface{}{"datasetId": "9"},
		})
		for k, name := range map[string]string{"c": "foo_bar_1", "a": "foo_bar", "b": "foo_bar_2"} {
			want := "${local.binding__dashboard_obj__dataset_" + name + "}"
			if got := out[k].(map[string]interface{})["datasetId"]; got != want {
				t.Errorf("%s: got %v, want %s", k, got, want)
			}
		}
	}
}

func TestFakeDuplicateLabelFails(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(
		bindingtest.Object{ID: "41000901", Label: "Dup"},
		bindingtest.Object{ID: "41000902", Label: "Dup"},
	))
	_, _, err := exportErr(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{
		"a": map[string]interface{}{"datasetId": "41000901"},
		"b": map[string]interface{}{"datasetId": "41000902"},
	})
	if err == nil || !strings.Contains(err.Error(), `share the label "Dup"`) {
		t.Fatalf("got %v, want duplicate label error", err)
	}

	// an unreferenced duplicate does not matter
	export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{"datasetId": "41000901"})
}

func TestFakeRawIdAndOidRejected(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	for _, oidLast := range []bool{false, true} {
		gen := newFakeGenerator(t, fake, NewKindSet(KindDataset, KindWorkspace))
		dsOid := oid.OID{Type: oid.TypeDataset, Id: "41000123"}
		gen.CollectOid(dsOid)
		if err := gen.Resolve(context.Background()); err != nil {
			t.Fatal(err)
		}
		if oidLast {
			gen.TryBindId(KindDataset, "41000123")
			gen.TryBindOid(dsOid)
		} else {
			gen.TryBindOid(dsOid)
			gen.TryBindId(KindDataset, "41000123")
		}
		if _, err := gen.GetBindings(); err == nil || !strings.Contains(err.Error(), "both by id and by oid") {
			t.Errorf("oidLast=%v: got %v, want id/oid error", oidLast, err)
		}
	}
}

func TestFakeUncollectedDatasetFails(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	gen := newFakeGenerator(t, fake, NewKindSet(KindDataset, KindWorkspace))
	if err := gen.Resolve(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ref, didBind := gen.TryBindId(KindDataset, "41000123"); didBind {
		t.Fatalf("bound uncollected id to %s", ref)
	}
	if _, err := gen.GetBindings(); err == nil {
		t.Fatal("expected error for uncollected dataset")
	}
}

// Only canonical positive ids are looked up; others and absent ids stay as-is.
func TestFakeInvalidAndAbsentCandidates(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	values := []string{"abc", "007", "0", "-5", "99999999999999999999", "1e3", " 41000123", "41009999", "41000123"}
	in := map[string]interface{}{"oids": []interface{}{"o:::dataset:abcdef", "o:::dataset:41009998/1"}}
	for i, v := range values {
		in[fmt.Sprint(i)] = map[string]interface{}{"datasetId": v}
	}
	out, _ := export(t, fake, NewKindSet(KindDataset, KindWorkspace), in)
	for i, v := range values {
		want := v
		if v == "41000123" {
			want = "${local.binding__dashboard_obj__dataset_logs}"
		}
		if got := out[fmt.Sprint(i)].(map[string]interface{})["datasetId"]; got != want {
			t.Errorf("%q: got %v, want %s", v, got, want)
		}
	}
	if got, want := out["oids"], []interface{}{"o:::dataset:abcdef", "o:::dataset:41009998/1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("oids: got %v, want %v", got, want)
	}
	if got, want := fake.DatasetFilterIDs(), [][]string{{"41000123", "41009998", "41009999"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("filters: got %v, want %v", got, want)
	}
}

func TestFakeRequestCount(t *testing.T) {
	for _, u := range []int{0, 1, 100, 101, 250} {
		var datasets []bindingtest.Object
		var refs []interface{}
		for i := 0; i < u; i++ {
			id := fmt.Sprint(42000000 + i)
			datasets = append(datasets, bindingtest.Object{ID: id, Label: "ds " + id})
			// repeats and both value forms of one id must not add requests
			refs = append(refs, map[string]interface{}{"datasetId": id}, map[string]interface{}{"dataset": id})
		}
		fake := bindingtest.New(t, fakeTenant(datasets...))
		out, b := export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{"refs": refs})
		if got, want := fake.Count(bindingtest.OpRestDatasets), (u+99)/100; got != want {
			t.Errorf("U=%d: got %d requests, want %d", u, got, want)
		}
		if len(b.Mappings) != u {
			t.Errorf("U=%d: got %d mappings", u, len(b.Mappings))
		}
		if raw, _ := json.Marshal(out); strings.Contains(string(raw), `"420`) {
			t.Errorf("U=%d: unbound ids remain", u)
		}
	}
}

func TestFakeOutputIndependentOfOrder(t *testing.T) {
	datasets := []bindingtest.Object{
		{ID: "41000801", Label: "Foo/Bar"},
		{ID: "41000802", Label: "foo bar"},
		{ID: "41000803", Label: "FOO-BAR"},
		{ID: "41000123", Label: "Logs"},
	}
	var first []byte
	for run := 0; run < 10; run++ {
		order := make([]bindingtest.Object, len(datasets))
		for i := range datasets {
			order[i] = datasets[(i+run)%len(datasets)]
		}
		if run%2 == 1 {
			for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
				order[i], order[j] = order[j], order[i]
			}
		}
		fake := bindingtest.New(t, fakeTenant(order...))
		in := map[string]interface{}{}
		for i, ds := range datasets {
			in[fmt.Sprint("k", i)] = map[string]interface{}{"datasetId": ds.ID}
		}
		out, b := export(t, fake, NewKindSet(KindDataset, KindWorkspace), in)
		got, err := json.Marshal([]interface{}{out, b})
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = got
		} else if string(got) != string(first) {
			t.Fatalf("run %d differs:\n%s\n%s", run, first, got)
		}
	}
}

func TestFakeNoDatasetRequestWithoutCandidates(t *testing.T) {
	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	export(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{"datasetId": "abc", "other": "41000123"})
	if n := fake.Count(bindingtest.OpRestDatasets); n != 0 {
		t.Errorf("got %d dataset requests, want 0", n)
	}
}

func TestFakeGeneratorFailures(t *testing.T) {
	failures := []bindingtest.Failure{bindingtest.FailUnauthorized, bindingtest.FailMalformed, bindingtest.FailNetwork}
	for _, f := range failures {
		fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
		fake.Fail("listWorkspaces", f)
		if _, err := NewGenerator(context.Background(), "dashboard", "Obj", fake.Client(), NewKindSet(KindDataset, KindWorkspace)); err == nil {
			t.Errorf("listWorkspaces/%d: expected error", f)
		}
	}
	for _, f := range failures {
		fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
		fake.Fail(bindingtest.OpRestDatasets, f)
		if _, _, err := exportErr(t, fake, NewKindSet(KindDataset, KindWorkspace), map[string]interface{}{"datasetId": "41000123"}); err == nil {
			t.Errorf("datasets/%d: expected error", f)
		}
	}

	fake := bindingtest.New(t, fakeTenant(bindingtest.Object{ID: "41000123", Label: "Logs"}))
	gen := newFakeGenerator(t, fake, NewKindSet(KindDataset, KindWorkspace))
	gen.CollectId(KindDataset, "41000123")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := gen.Resolve(ctx); err == nil {
		t.Error("canceled: expected error")
	}
}
