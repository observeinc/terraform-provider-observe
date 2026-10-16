package observe

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

type opalVersionTestReader struct {
	state map[string]interface{}
	raw   cty.Value
}

func (r opalVersionTestReader) Get(key string) interface{} {
	return r.state[key]
}

func (r opalVersionTestReader) GetOk(key string) (interface{}, bool) {
	value, ok := r.state[key]
	return value, ok && value != ""
}

func (r opalVersionTestReader) GetRawConfig() cty.Value {
	return r.raw
}

func TestResolveOpalVersion(t *testing.T) {
	earliest := "v2021_01_01"
	latest := "v2026_09_30"

	tests := []struct {
		name       string
		config     map[string]cty.Value
		state      map[string]interface{}
		defaultPin *string
		want       *gql.OpalVersion
	}{
		{
			name:       "resource config overrides provider default",
			config:     map[string]cty.Value{"opal_version": cty.StringVal(earliest)},
			state:      map[string]interface{}{"opal_version": latest},
			defaultPin: &latest,
			want:       opalVersionPtr(gql.OpalVersionV20210101),
		},
		{
			name:       "provider default overrides computed state",
			config:     map[string]cty.Value{},
			state:      map[string]interface{}{"opal_version": earliest},
			defaultPin: &latest,
			want:       opalVersionPtr(gql.OpalVersionV20260930),
		},
		{
			name:       "unknown resource config does not fall back to provider default",
			config:     map[string]cty.Value{"opal_version": cty.UnknownVal(cty.String)},
			state:      map[string]interface{}{"opal_version": earliest},
			defaultPin: &latest,
		},
		{
			name:   "computed state is retained without configured pin",
			config: map[string]cty.Value{},
			state:  map[string]interface{}{"opal_version": earliest},
			want:   opalVersionPtr(gql.OpalVersionV20210101),
		},
		{
			name:   "new object without a pin omits version",
			config: map[string]cty.Value{},
			state:  map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := opalVersionTestReader{
				state: tt.state,
				raw:   cty.ObjectVal(tt.config),
			}
			got := resolveOpalVersion(tt.defaultPin, reader)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("resolveOpalVersion() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != *tt.want {
				t.Fatalf("resolveOpalVersion() = %v, want %q", got, *tt.want)
			}
		})
	}
}

func TestOpalVersionSchema(t *testing.T) {
	field := opalVersionSchema()
	if !field.Optional || !field.Computed || field.Type != schema.TypeString {
		t.Fatalf("opalVersionSchema() = %#v, want optional+computed string", field)
	}

	for _, value := range []string{"v2021_01_01", "v2026_09_30"} {
		if diagnostics := field.ValidateDiagFunc(value, cty.Path{}); diagnostics.HasError() {
			t.Fatalf("opalVersionSchema() rejected %q: %v", value, diagnostics)
		}
	}
	if diagnostics := field.ValidateDiagFunc("latest", cty.Path{}); !diagnostics.HasError() {
		t.Fatal("opalVersionSchema() accepted invalid value")
	}
}

func TestProviderHasDefaultOpalVersion(t *testing.T) {
	field, ok := Provider().Schema["default_opal_version"]
	if !ok {
		t.Fatal("provider schema does not contain default_opal_version")
	}
	if !field.Optional || field.Type != schema.TypeString {
		t.Fatalf("default_opal_version = %#v, want optional string", field)
	}
}

func TestNewQueryUsesResolvedOpalVersion(t *testing.T) {
	defaultPin := "v2026_09_30"
	reader := opalVersionTestReader{
		state: map[string]interface{}{
			"inputs": map[string]interface{}{"source": "o:::dataset:41000001"},
			"stage": []interface{}{
				map[string]interface{}{"pipeline": "filter true"},
			},
		},
		raw: cty.ObjectVal(map[string]cty.Value{}),
	}

	query, diagnostics := newQuery(reader, &defaultPin)
	if diagnostics.HasError() {
		t.Fatalf("newQuery() diagnostics: %v", diagnostics)
	}
	if query.OpalVersion == nil || *query.OpalVersion != gql.OpalVersionV20260930 {
		t.Fatalf("newQuery().OpalVersion = %v, want %q", query.OpalVersion, gql.OpalVersionV20260930)
	}
}

func TestGraphQLWorkspaceObjectsExposeOpalVersion(t *testing.T) {
	resources := map[string]*schema.Resource{
		"dataset":                    resourceDataset(),
		"monitor":                    resourceMonitor(),
		"monitor_v2":                 resourceMonitorV2(),
		"dashboard":                  resourceDashboard(),
		"worksheet":                  resourceWorksheet(),
		"log_derived_metric_dataset": resourceLogDerivedMetricDataset(),
	}
	for name, resource := range resources {
		t.Run("resource_"+name, func(t *testing.T) {
			field, ok := resource.Schema["opal_version"]
			if !ok {
				t.Fatal("schema does not contain opal_version")
			}
			if !field.Optional || !field.Computed {
				t.Fatalf("opal_version = %#v, want optional+computed", field)
			}
			if resource.CustomizeDiff == nil {
				t.Fatal("resource does not schedule provider-default OPAL version changes")
			}
		})
	}

	dataSources := map[string]*schema.Resource{
		"dataset":    dataSourceDataset(),
		"monitor":    dataSourceMonitor(),
		"monitor_v2": dataSourceMonitorV2(),
		"dashboard":  dataSourceDashboard(),
		"worksheet":  dataSourceWorksheet(),
	}
	for name, dataSource := range dataSources {
		t.Run("data_source_"+name, func(t *testing.T) {
			field, ok := dataSource.Schema["opal_version"]
			if !ok {
				t.Fatal("schema does not contain opal_version")
			}
			if !field.Computed || field.Optional {
				t.Fatalf("opal_version = %#v, want computed-only", field)
			}
		})
	}
}

func TestObjectInputBuildersUseProviderDefaultOpalVersion(t *testing.T) {
	defaultPin := "v2026_09_30"

	t.Run("dashboard", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
			"name": "dashboard",
		})
		input, diagnostics := newDashboardConfig(data, &defaultPin)
		if diagnostics.HasError() {
			t.Fatalf("newDashboardConfig() diagnostics: %v", diagnostics)
		}
		if input.OpalVersion == nil || *input.OpalVersion != gql.OpalVersionV20260930 {
			t.Fatalf("OpalVersion = %v, want %q", input.OpalVersion, gql.OpalVersionV20260930)
		}
	})

	t.Run("worksheet", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceWorksheet().Schema, map[string]interface{}{
			"name": "worksheet",
		})
		input, diagnostics := newWorksheetConfig(data, &defaultPin)
		if diagnostics.HasError() {
			t.Fatalf("newWorksheetConfig() diagnostics: %v", diagnostics)
		}
		if input.OpalVersion == nil || *input.OpalVersion != gql.OpalVersionV20260930 {
			t.Fatalf("OpalVersion = %v, want %q", input.OpalVersion, gql.OpalVersionV20260930)
		}
	})

	t.Run("log derived metric", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceLogDerivedMetricDataset().Schema, map[string]interface{}{
			"metric_name":   "errors",
			"input":         "o:::dataset:41000001",
			"shaping_query": "filter true",
			"aggregation": []interface{}{
				map[string]interface{}{"function": "count"},
			},
		})
		input, diagnostics := newLogDerivedMetricDefinitionInput(data, &defaultPin)
		if diagnostics.HasError() {
			t.Fatalf("newLogDerivedMetricDefinitionInput() diagnostics: %v", diagnostics)
		}
		if input.OpalVersion == nil || *input.OpalVersion != gql.OpalVersionV20260930 {
			t.Fatalf("OpalVersion = %v, want %q", input.OpalVersion, gql.OpalVersionV20260930)
		}
	})
}

func TestRESTWorkspaceObjectsExposeOpalVersion(t *testing.T) {
	resources := map[string]*schema.Resource{
		"ingest_route":         resourceIngestRoute(),
		"drop_filter":          resourceDropFilter(),
		"dataset_query_filter": resourceDatasetQueryFilter(),
	}
	for name, resource := range resources {
		t.Run(name, func(t *testing.T) {
			field, ok := resource.Schema["opal_version"]
			if !ok {
				t.Fatal("schema does not contain opal_version")
			}
			if !field.Optional || !field.Computed {
				t.Fatalf("opal_version = %#v, want optional+computed", field)
			}
			if resource.CustomizeDiff == nil {
				t.Fatal("resource does not schedule provider-default OPAL version changes")
			}
		})
	}
}

func TestRESTRequestBuildersUseProviderDefaultOpalVersion(t *testing.T) {
	defaultPin := "v2026_09_30"

	t.Run("ingest route", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{
			"type":           "otellogs",
			"pipeline":       "filter true",
			"destination_id": "41000001",
		})
		request := ingestRouteRequestFromResourceData(data, &defaultPin)
		if request.OpalVersion == nil || *request.OpalVersion != defaultPin {
			t.Fatalf("OpalVersion = %v, want %q", request.OpalVersion, defaultPin)
		}
	})

	t.Run("drop filter create", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceDropFilter().Schema, map[string]interface{}{
			"name":           "drop",
			"pipeline":       "filter true",
			"drop_rate":      1.0,
			"source_dataset": "o:::dataset:41000001",
		})
		request, diagnostics := newIngestFilterCreateRequest(data, &defaultPin)
		if diagnostics.HasError() {
			t.Fatalf("newIngestFilterCreateRequest() diagnostics: %v", diagnostics)
		}
		if request.OpalVersion == nil || *request.OpalVersion != defaultPin {
			t.Fatalf("OpalVersion = %v, want %q", request.OpalVersion, defaultPin)
		}
	})

	t.Run("dataset query filter", func(t *testing.T) {
		data := schema.TestResourceDataRaw(t, resourceDatasetQueryFilter().Schema, map[string]interface{}{
			"dataset": "o:::dataset:41000001",
			"label":   "errors",
			"filter":  "severity = \"error\"",
		})
		request, diagnostics := datasetQueryFilterDefinitionFromResourceData(data, &defaultPin)
		if diagnostics.HasError() {
			t.Fatalf("datasetQueryFilterDefinitionFromResourceData() diagnostics: %v", diagnostics)
		}
		if request.OpalVersion == nil || *request.OpalVersion != defaultPin {
			t.Fatalf("OpalVersion = %v, want %q", request.OpalVersion, defaultPin)
		}
	})
}
