package observe

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/binding/bindingtest"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

var updateGoldens = flag.Bool("update-bindings", false, "rewrite binding export golden files")

// datasetLookupOp is the request that resolves dataset labels during export.
const datasetLookupOp = bindingtest.OpRestDatasets

func exportTenant() bindingtest.Tenant {
	return bindingtest.Tenant{
		Workspace: bindingtest.Object{ID: "41000001", Label: "Default"},
		Datasets: []bindingtest.Object{
			{ID: "41000123", Label: "Kubernetes/Container Logs"},
			{ID: "41000200", Label: "usage/Monitor Messages"},
			{ID: "41000300", Label: "Tracing/Span"},
			{ID: "41000500", Label: "Shared Id Dataset"},
			{ID: "41000600", Label: "Unreferenced"},
		},
		Worksheets: []bindingtest.Object{
			{ID: "41000201", Label: "My Worksheet"},
			{ID: "41000500", Label: "Shared Id Worksheet"},
		},
		Users: []bindingtest.User{
			{ID: 41000100, Email: "basic@example.com", Label: "Basic User"},
		},
		MonitorV2Actions: []bindingtest.Object{
			{ID: "41000700", Label: "Page Oncall"},
		},
	}
}

// exporter seeds a data source with fields, runs its binding generation and
// reports the published fields. jsonFields hold JSON documents as strings.
type exporter struct {
	resource   func() *schema.Resource
	fields     map[string]interface{}
	jsonFields []string
	generate   func(context.Context, *schema.ResourceData, *observe.Client) error
}

var exporters = map[string]exporter{
	"dashboard": {
		resource: dataSourceDashboard,
		fields: map[string]interface{}{
			"stages": `[{"id":"stage-1","input":[{"datasetId":"41000123","inputName":"logs","inputRole":"Data"},` +
				`{"datasetId":"41009999","inputName":"gone"},{"datasetId":"abc","inputName":"bad"}],` +
				`"pipeline":"filter true","params":{"sourceDatasetId":"o:::dataset:41000200/1700000000000"}}]`,
			"parameters": `[{"id":"ds_param","defaultValue":{"datasetref":{"datasetId":"41000300"}},` +
				`"valueKind":{"type":"DATASETREF","keyForDatasetId":"41000123"}}]`,
			"parameter_values": `[{"id":"ds_param","value":{"datasetref":{"datasetId":"41000300"}}},` +
				`{"id":"user","value":{"userId":"41000100"}}]`,
			"layout": `{"gridLayout":{"sections":[{"items":[{"card":{"cardType":"stage","datasetId":"41000123","stageId":"stage-1"}},` +
				`{"card":{"id":"41000500"}}]}]},` +
				`"stageListLayout":{"inputs":[["o:::dataset:41000200","o:::dataset:abcdef","o:::user:41000100","o:::dataset:41009999"]]},` +
				`"workspaceId":"41000001","otherWorkspace":{"workspaceId":"41000002"},"count":"41000123"}`,
		},
		jsonFields: []string{"stages", "parameters", "parameter_values", "layout"},
		generate: func(ctx context.Context, data *schema.ResourceData, c *observe.Client) error {
			return generateDashboardBindings(ctx, &gql.Dashboard{Name: "Ops Overview", WorkspaceId: "41000001"}, data, c)
		},
	},
	"worksheet": {
		resource: dataSourceWorksheet,
		fields: map[string]interface{}{
			"queries": `[{"id":"q1","layout":{"id":"41000300"},"stages":[{"input":[{"datasetId":"41000123","inputName":"a"},` +
				`{"datasetId":"o:::dataset:41000200","inputName":"b"},{"datasetId":"41009999","inputName":"c"}],"stageID":"s1"}]},` +
				`{"id":"41000201","stages":[]}]`,
		},
		jsonFields: []string{"queries", "_bindings"},
		generate: func(ctx context.Context, data *schema.ResourceData, c *observe.Client) error {
			return generateWorksheetBindings(ctx, &gql.Worksheet{Label: "Log Explorer", WorkspaceId: "41000001"}, data, c)
		},
	},
	"monitor": {
		resource: dataSourceMonitor,
		fields: map[string]interface{}{
			"inputs": map[string]interface{}{
				"logs":  "o:::dataset:41000123",
				"spans": "o:::dataset:41000300/1700000000000",
				"raw":   "41000200",
				"gone":  "o:::dataset:41009999",
				"user":  "o:::user:41000100",
			},
		},
		jsonFields: []string{"_bindings"},
		generate: func(ctx context.Context, data *schema.ResourceData, c *observe.Client) error {
			return generateMonitorBindings(ctx, &gql.Monitor{Name: "Error Rate", WorkspaceId: "41000001"}, data, c)
		},
	},
	"monitor_v2": {
		resource: dataSourceMonitorV2,
		fields: map[string]interface{}{
			"inputs": map[string]interface{}{
				"logs":  "o:::dataset:41000123",
				"spans": "o:::dataset:41000300",
				"gone":  "o:::dataset:41009999",
			},
			"actions": []interface{}{
				map[string]interface{}{"oid": "o:::monitorv2action:41000700", "levels": []interface{}{"error"}},
				map[string]interface{}{"oid": "o:::monitorv2action:41000799"},
			},
		},
		jsonFields: []string{"_bindings"},
		generate: func(ctx context.Context, data *schema.ResourceData, c *observe.Client) error {
			return generateMonitorV2Bindings(ctx, &gql.MonitorV2{Name: "Latency SLO", WorkspaceId: "41000001"}, data, c)
		},
	},
	"monitor_v2_action": {
		resource: dataSourceMonitorV2Action,
		fields: map[string]interface{}{
			"email": []interface{}{
				map[string]interface{}{
					"users":     []interface{}{"o:::user:41000100", "o:::user:41000199"},
					"addresses": []interface{}{"oncall@example.com"},
					"subject":   "dataset 41000123",
				},
			},
		},
		jsonFields: []string{"_bindings"},
		generate: func(ctx context.Context, data *schema.ResourceData, c *observe.Client) error {
			return generateMonitorV2ActionBindings(ctx, &gql.MonitorV2Action{Name: "Page Oncall", WorkspaceId: "41000001"}, data, c)
		},
	},
}

func (e exporter) seed(t *testing.T) *schema.ResourceData {
	t.Helper()
	data := schema.TestResourceDataRaw(t, e.resource().Schema, map[string]interface{}{})
	for field, value := range e.fields {
		if err := data.Set(field, value); err != nil {
			t.Fatalf("seeding %s: %s", field, err)
		}
	}
	return data
}

// published returns every field the exporter writes, decoding JSON fields.
func (e exporter) published(t *testing.T, data *schema.ResourceData) map[string]interface{} {
	t.Helper()
	out := map[string]interface{}{"workspace": data.Get("workspace")}
	for field := range e.fields {
		out[field] = data.Get(field)
	}
	for _, field := range e.jsonFields {
		raw := data.Get(field).(string)
		if raw == "" {
			out[field] = nil
			continue
		}
		var decoded interface{}
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatalf("field %s is not JSON: %s", field, err)
		}
		out[field] = decoded
	}
	return out
}

func checkGolden(t *testing.T, name string, got interface{}) {
	t.Helper()
	path := filepath.Join("testdata", "bindings", name+".golden.json")
	gotBytes, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gotBytes = append(gotBytes, '\n')
	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, gotBytes, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden (run with -update-bindings to create): %s", err)
	}
	if string(want) != string(gotBytes) {
		t.Fatalf("%s mismatch\n--- want\n%s\n--- got\n%s", path, want, gotBytes)
	}
}

func TestBindingExportGolden(t *testing.T) {
	for name, e := range exporters {
		t.Run(name, func(t *testing.T) {
			fake := bindingtest.New(t, exportTenant())
			data := e.seed(t)
			if err := e.generate(context.Background(), data, fake.Client()); err != nil {
				t.Fatal(err)
			}
			published := e.published(t, data)
			checkGolden(t, name, published)

			bindings := published["_bindings"]
			if name == "dashboard" {
				bindings = published["layout"].(map[string]interface{})["bindings"]
			}
			raw, err := json.Marshal(bindings)
			if err != nil {
				t.Fatal(err)
			}
			if undefined, err := bindingtest.UndefinedLocals(published, raw); err != nil || len(undefined) != 0 {
				t.Errorf("references without bindings: %v %v", undefined, err)
			}

			// all candidates of one object resolve in one request
			want := 1
			if name == "monitor_v2_action" {
				want = 0
			}
			if got := fake.Count(bindingtest.OpRestDatasets); got != want {
				t.Errorf("got %d dataset requests, want %d", got, want)
			}
		})
	}
}

func TestBindingExportRejectsAmbiguousDatasets(t *testing.T) {
	tenant := exportTenant()
	tenant.Datasets = append(tenant.Datasets, bindingtest.Object{ID: "41000124", Label: "Kubernetes/Container Logs"})
	cases := map[string]struct {
		tenant bindingtest.Tenant
		stages string
	}{
		"raw id and oid":  {exportTenant(), `[{"input":[{"datasetId":"41000123"}],"params":{"x":"o:::dataset:41000123"}}]`},
		"duplicate label": {tenant, `[{"input":[{"datasetId":"41000123"},{"datasetId":"41000124"}]}]`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := bindingtest.New(t, tc.tenant)
			e := exporters["dashboard"]
			data := e.seed(t)
			if err := data.Set("stages", tc.stages); err != nil {
				t.Fatal(err)
			}
			before := e.published(t, data)
			if err := e.generate(context.Background(), data, fake.Client()); err == nil {
				t.Fatal("expected export to fail")
			}
			if after := e.published(t, data); !reflect.DeepEqual(before, after) {
				t.Fatalf("failed export changed state\nbefore: %#v\nafter:  %#v", before, after)
			}
		})
	}
}

func TestBindingExportActionMakesNoDatasetRequests(t *testing.T) {
	fake := bindingtest.New(t, exportTenant())
	e := exporters["monitor_v2_action"]
	if err := e.generate(context.Background(), e.seed(t), fake.Client()); err != nil {
		t.Fatal(err)
	}
	if n := fake.Count(bindingtest.OpRestDatasets); n != 0 {
		t.Errorf("got %d dataset requests, want 0", n)
	}
}

func TestBindingExportFailures(t *testing.T) {
	failures := map[string]bindingtest.Failure{
		"unauthorized": bindingtest.FailUnauthorized,
		"malformed":    bindingtest.FailMalformed,
		"network":      bindingtest.FailNetwork,
	}
	ops := map[string][]string{
		"dashboard":         {"listWorkspaces", datasetLookupOp},
		"worksheet":         {"listWorkspaces", datasetLookupOp},
		"monitor":           {"listWorkspaces", datasetLookupOp},
		"monitor_v2":        {"listWorkspaces", datasetLookupOp, "searchMonitorV2Action"},
		"monitor_v2_action": {"listWorkspaces", "listUsers"},
	}
	for name, e := range exporters {
		for _, op := range ops[name] {
			for fname, f := range failures {
				t.Run(name+"/"+op+"/"+fname, func(t *testing.T) {
					fake := bindingtest.New(t, exportTenant())
					fake.Fail(op, f)
					data := e.seed(t)
					before := e.published(t, data)
					if err := e.generate(context.Background(), data, fake.Client()); err == nil {
						t.Fatal("expected export to fail")
					}
					if after := e.published(t, data); !reflect.DeepEqual(before, after) {
						t.Fatalf("failed export changed state\nbefore: %#v\nafter:  %#v", before, after)
					}
				})
			}
		}
	}
}

// Labels found by one export are reused by later exports on the same client;
// absent ids are looked up again.
func TestBindingExportReusesLabels(t *testing.T) {
	fake := bindingtest.New(t, exportTenant())
	c := fake.Client()
	e := exporters["monitor_v2"]
	for i, want := range []int{1, 2} {
		if err := e.generate(context.Background(), e.seed(t), c); err != nil {
			t.Fatal(err)
		}
		if got := fake.Count(bindingtest.OpRestDatasets); got != want {
			t.Errorf("export %d: got %d total dataset requests, want %d", i, got, want)
		}
	}
	if got, want := fake.DatasetFilterIDs()[1], []string{"41009999"}; !reflect.DeepEqual(got, want) {
		t.Errorf("warm export looked up %v, want %v", got, want)
	}
}
