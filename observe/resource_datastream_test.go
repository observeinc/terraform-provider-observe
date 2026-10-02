package observe

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var (
	datastreamConfigPreamble = `
	resource "observe_datastream" "test" {
		workspace = data.observe_workspace.default.oid
		name      = "%[1]s"
	}`
)

func TestDatastreamTypeSchema(t *testing.T) {
	resource := resourceDatastream()
	typeSchema, ok := resource.Schema["type"]
	if !ok {
		t.Fatal("type schema is missing")
	}
	if !typeSchema.Optional || !typeSchema.Computed || typeSchema.ForceNew {
		t.Errorf("type schema = %#v, want optional and computed, not ForceNew", typeSchema)
	}
	if resource.CustomizeDiff == nil {
		t.Error("datastream resource has no CustomizeDiff to reject type changes")
	}
	if forceDestroy, ok := resource.Schema["force_destroy"]; !ok {
		t.Error("force_destroy schema is missing")
	} else if forceDestroy.Type != schema.TypeBool || !forceDestroy.Optional || forceDestroy.ForceNew || forceDestroy.Default != nil {
		t.Errorf("force_destroy schema = %#v, want optional bool without default or ForceNew", forceDestroy)
	}
	for _, typeName := range []string{"Prometheus", "OtelLogs", "OtelMetrics", "K8sEntity", "OtelTrace"} {
		if diags := typeSchema.ValidateDiagFunc(typeName, nil); diags.HasError() {
			t.Errorf("type %q rejected: %v", typeName, diags)
		}
	}
	if diags := typeSchema.ValidateDiagFunc("invalid", nil); !diags.HasError() {
		t.Error("invalid type accepted")
	}
}

func TestNewDatastreamConfigLeavesTypeUnset(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
		"name": "untyped",
	})

	config, diags := newDatastreamConfig(data)
	if diags.HasError() {
		t.Fatalf("new datastream config: %v", diags)
	}
	if config.DirectWrite != nil {
		t.Errorf("direct write = %#v, want nil", config.DirectWrite)
	}
}

func TestNewDatastreamConfigMapsTypeToDirectWrite(t *testing.T) {
	testCases := []struct {
		typeName string
		matches  func(*gql.DatastreamDirectWriteInput) bool
	}{
		{"Prometheus", func(input *gql.DatastreamDirectWriteInput) bool {
			return input.Prometheus != nil && *input.Prometheus && input.OtelLogs == nil && input.OtelMetrics == nil && input.K8sEntity == nil && input.OtelTrace == nil
		}},
		{"OtelLogs", func(input *gql.DatastreamDirectWriteInput) bool {
			return input.Prometheus == nil && input.OtelLogs != nil && *input.OtelLogs && input.OtelMetrics == nil && input.K8sEntity == nil && input.OtelTrace == nil
		}},
		{"OtelMetrics", func(input *gql.DatastreamDirectWriteInput) bool {
			return input.Prometheus == nil && input.OtelLogs == nil && input.OtelMetrics != nil && *input.OtelMetrics && input.K8sEntity == nil && input.OtelTrace == nil
		}},
		{"K8sEntity", func(input *gql.DatastreamDirectWriteInput) bool {
			return input.Prometheus == nil && input.OtelLogs == nil && input.OtelMetrics == nil && input.K8sEntity != nil && *input.K8sEntity && input.OtelTrace == nil
		}},
		{"OtelTrace", func(input *gql.DatastreamDirectWriteInput) bool {
			return input.Prometheus == nil && input.OtelLogs == nil && input.OtelMetrics == nil && input.K8sEntity == nil && input.OtelTrace != nil && *input.OtelTrace
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.typeName, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
				"name": "typed",
				"type": testCase.typeName,
			})
			config, diags := newDatastreamConfig(data)
			if diags.HasError() {
				t.Fatalf("new datastream config: %v", diags)
			}
			if config.DirectWrite == nil || !testCase.matches(config.DirectWrite) {
				t.Errorf("direct write = %#v, want only %s enabled", config.DirectWrite, testCase.typeName)
			}
		})
	}
}

func TestNewDatastreamConfigRejectsUnsupportedType(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
		"name": "typed",
		"type": "unsupported",
	})

	_, diags := newDatastreamConfig(data)
	if !diags.HasError() {
		t.Fatal("expected unsupported type diagnostic")
	}
}

func TestDatastreamToResourceDataMapsDirectWriteType(t *testing.T) {
	testCases := []struct {
		typeName    string
		directWrite *gql.DatastreamDirectWrite
	}{
		{"Prometheus", &gql.DatastreamDirectWrite{Prometheus: &gql.DatastreamDirectWritePrometheusDatastreamDirectWriteInfoPrometheus{}}},
		{"OtelLogs", &gql.DatastreamDirectWrite{OtelLogs: &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{}}},
		{"OtelMetrics", &gql.DatastreamDirectWrite{OtelMetrics: &gql.DatastreamDirectWriteOtelMetricsDatastreamDirectWriteInfo{}}},
		{"K8sEntity", &gql.DatastreamDirectWrite{K8sEntity: &gql.DatastreamDirectWriteK8sEntityDatastreamDirectWriteInfo{}}},
		{"OtelTrace", &gql.DatastreamDirectWrite{OtelTrace: &gql.DatastreamDirectWriteOtelTraceDatastreamDirectWriteInfoOtelTrace{}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.typeName, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{})
			diags := datastreamToResourceData(&gql.Datastream{
				Id:          "41030001",
				Name:        "typed",
				WorkspaceId: "41030002",
				DirectWrite: testCase.directWrite,
			}, data)
			if diags.HasError() {
				t.Fatalf("map datastream: %v", diags)
			}
			if got := data.Get("type"); got != testCase.typeName {
				t.Errorf("type = %q, want %q", got, testCase.typeName)
			}
		})
	}
}

func TestDatastreamToResourceDataLeavesUntypedTypeUnset(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "untyped",
		WorkspaceId: "41030002",
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "" {
		t.Errorf("type = %q, want unset", got)
	}
}

func TestDatastreamToResourceDataPreservesTypeWhenDirectWriteIsOmitted(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
		"type": "OtelLogs",
	})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "legacy",
		WorkspaceId: "41030002",
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "OtelLogs" {
		t.Errorf("type = %q, want existing value OtelLogs", got)
	}
}

func TestDatastreamToResourceDataPreservesTypeForMultiTypeDatastream(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
		"type": "OtelLogs",
	})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "legacy",
		WorkspaceId: "41030002",
		DirectWrite: &gql.DatastreamDirectWrite{
			OtelLogs:    &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{},
			OtelMetrics: &gql.DatastreamDirectWriteOtelMetricsDatastreamDirectWriteInfo{},
		},
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "OtelLogs" {
		t.Errorf("type = %q, want existing value OtelLogs", got)
	}
}

func TestDatastreamToResourceDataPreservesTypeForEmptyDirectWrite(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{
		"type": "OtelLogs",
	})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "legacy",
		WorkspaceId: "41030002",
		DirectWrite: &gql.DatastreamDirectWrite{},
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "OtelLogs" {
		t.Errorf("type = %q, want existing value OtelLogs", got)
	}
}

func TestDatastreamToResourceDataUsesDirectWriteDataset(t *testing.T) {
	testCases := []struct {
		name        string
		directWrite *gql.DatastreamDirectWrite
		wantDataset string
	}{
		{"Prometheus", &gql.DatastreamDirectWrite{Prometheus: &gql.DatastreamDirectWritePrometheusDatastreamDirectWriteInfoPrometheus{DatasetId: "41030011"}}, "41030011"},
		{"OtelLogs", &gql.DatastreamDirectWrite{OtelLogs: &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{DatasetId: "41030012"}}, "41030012"},
		{"OtelMetrics", &gql.DatastreamDirectWrite{OtelMetrics: &gql.DatastreamDirectWriteOtelMetricsDatastreamDirectWriteInfo{DatasetId: "41030013"}}, "41030013"},
		{"K8sEntity", &gql.DatastreamDirectWrite{K8sEntity: &gql.DatastreamDirectWriteK8sEntityDatastreamDirectWriteInfo{DatasetId: "41030014"}}, "41030014"},
		{"OtelTrace", &gql.DatastreamDirectWrite{OtelTrace: &gql.DatastreamDirectWriteOtelTraceDatastreamDirectWriteInfoOtelTrace{SpanDatasetId: "41030015"}}, "41030015"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{})
			diags := datastreamToResourceData(&gql.Datastream{
				Id:          "41030001",
				Name:        "typed",
				WorkspaceId: "41030002",
				DirectWrite: testCase.directWrite,
			}, data)
			if diags.HasError() {
				t.Fatalf("map datastream: %v", diags)
			}
			if got := data.Get("dataset"); got != oid.DatasetOid(testCase.wantDataset).String() {
				t.Errorf("dataset = %q, want %q", got, oid.DatasetOid(testCase.wantDataset).String())
			}
		})
	}
}

func TestDatastreamToResourceDataPrefersTopLevelDataset(t *testing.T) {
	topLevelDataset := "41030020"
	data := schema.TestResourceDataRaw(t, resourceDatastream().Schema, map[string]interface{}{})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "typed",
		WorkspaceId: "41030002",
		DatasetId:   &topLevelDataset,
		DirectWrite: &gql.DatastreamDirectWrite{
			OtelLogs: &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{DatasetId: "41030021"},
		},
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("dataset"); got != oid.DatasetOid(topLevelDataset).String() {
		t.Errorf("dataset = %q, want top-level dataset %q", got, oid.DatasetOid(topLevelDataset).String())
	}
}

func TestDatastreamToResourceDataSupportsTypedDataSourceSchema(t *testing.T) {
	data := schema.TestResourceDataRaw(t, dataSourceDatastream().Schema, map[string]interface{}{})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "typed",
		WorkspaceId: "41030002",
		DirectWrite: &gql.DatastreamDirectWrite{
			OtelLogs: &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{},
		},
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "OtelLogs" {
		t.Errorf("type = %q, want OtelLogs", got)
	}
}

func TestDatastreamToResourceDataLeavesMultiTypeDataSourceReadable(t *testing.T) {
	data := schema.TestResourceDataRaw(t, dataSourceDatastream().Schema, map[string]interface{}{})
	diags := datastreamToResourceData(&gql.Datastream{
		Id:          "41030001",
		Name:        "invalid",
		WorkspaceId: "41030002",
		DirectWrite: &gql.DatastreamDirectWrite{
			OtelLogs:    &gql.DatastreamDirectWriteOtelLogsDatastreamDirectWriteInfo{},
			OtelMetrics: &gql.DatastreamDirectWriteOtelMetricsDatastreamDirectWriteInfo{},
		},
	}, data)
	if diags.HasError() {
		t.Fatalf("map datastream: %v", diags)
	}
	if got := data.Get("type"); got != "" {
		t.Errorf("type = %q, want unset", got)
	}
}

func TestAccObserveDatastreamNameValidationTooLong(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s%s"  # exceeds MaxNameLength
					icon_url  = "test"
				}
				`, randomPrefix, strings.Repeat("a", MaxNameLength)),
				ExpectError: regexp.MustCompile("expected length of name to be.*"),
			},
		},
	})
}

func TestAccObserveDatastreamNameValidationInvalidCharacter(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s with colon :"
					icon_url  = "test"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile("expected value of name to not contain.*"),
			},
		},
	})
}

func TestAccObserveDatastreamCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					icon_url  = "test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_datastream.example", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_datastream.example", "dataset"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-bis"
					icon_url  = "changed"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix+"-bis"),
					resource.TestCheckResourceAttr("observe_datastream.example", "icon_url", "changed"),
					resource.TestCheckResourceAttrSet("observe_datastream.example", "dataset"),
				),
			},
		},
	})
}

func TestDatastreamTypeChangeError(t *testing.T) {
	for _, testCase := range []struct {
		name, oldType, newType string
		wantErr                bool
	}{
		{name: "unchanged typed", oldType: "OtelLogs", newType: "OtelLogs"},
		{name: "unchanged any", oldType: "", newType: ""},
		{name: "any to typed", oldType: "", newType: "OtelLogs", wantErr: true},
		{name: "typed to typed", oldType: "OtelLogs", newType: "OtelMetrics", wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := datastreamTypeChangeError("41084453", testCase.oldType, testCase.newType)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("datastreamTypeChangeError(%q, %q) = %v, want error %v", testCase.oldType, testCase.newType, err, testCase.wantErr)
			}
		})
	}
}

func TestDatastreamTypeChangeDiff(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	for _, testCase := range []struct {
		name              string
		stateType         string
		config            map[string]interface{}
		wantErr           string
		wantDiff, wantNew bool
	}{
		{name: "any to typed", config: map[string]interface{}{"type": "OtelLogs"}, wantErr: "force_destroy = true"},
		{name: "typed to typed", stateType: "OtelLogs", config: map[string]interface{}{"type": "OtelMetrics"}, wantErr: "type cannot be changed"},
		{name: "any to typed with force_destroy", config: map[string]interface{}{"type": "OtelLogs", "force_destroy": true}, wantDiff: true, wantNew: true},
		{name: "typed to typed with force_destroy", stateType: "OtelLogs", config: map[string]interface{}{"type": "OtelMetrics", "force_destroy": true}, wantDiff: true, wantNew: true},
		{name: "unknown force_destroy", config: map[string]interface{}{"type": "OtelLogs", "force_destroy": unknown}, wantErr: "must be known"},
		{name: "force_destroy only", stateType: "OtelLogs", config: map[string]interface{}{"type": "OtelLogs", "force_destroy": true}, wantDiff: true},
		{name: "type removed", stateType: "OtelLogs", config: map[string]interface{}{}},
		{name: "unknown type", config: map[string]interface{}{"type": unknown}, wantDiff: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := &terraform.InstanceState{ID: "41084453", Attributes: map[string]string{
				"id": "41084453", "name": "audit", "workspace": "o:::workspace:41000001",
				"dataset": "o:::dataset:41084454", "oid": "o:::datastream:41084453",
			}}
			if testCase.stateType != "" {
				state.Attributes["type"] = testCase.stateType
			}
			config := map[string]interface{}{"name": "audit", "workspace": "o:::workspace:41000001"}
			for k, v := range testCase.config {
				config[k] = v
			}
			diff, err := resourceDatastream().Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("err = %v, want error containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotDiff := diff != nil; gotDiff != testCase.wantDiff {
				t.Fatalf("diff = %v, want diff %v", diff, testCase.wantDiff)
			}
			if diff != nil && diff.RequiresNew() != testCase.wantNew {
				t.Errorf("RequiresNew = %v, want %v", diff.RequiresNew(), testCase.wantNew)
			}
		})
	}
}
