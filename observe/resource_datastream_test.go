package observe

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	if !typeSchema.Optional || !typeSchema.Computed || !typeSchema.ForceNew {
		t.Errorf("type schema = %#v, want optional, computed, and ForceNew", typeSchema)
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
