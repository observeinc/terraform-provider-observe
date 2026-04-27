package observe

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func TestAccObserveLogDerivedMetricDatasetCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	config := fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					description = "test log-derived metric"

					metric_name = "error_count"
					metric_type = "gauge"
					unit        = "1"
					interval    = "1m"

					input = observe_datastream.test.dataset

					aggregation {
						function = "count"
					}
				}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "workspace"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "test log-derived metric"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "error_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_type", "gauge"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "unit", "1"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "oid"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDatasetDefaultsDoNotDrift(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	config := fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid

					metric_name = "drift_check_count"

					input = observe_datastream.test.dataset

					aggregation {
						function = "count"
					}
				}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "drift_check_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDatasetUpdate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					description = "initial description"

					metric_name = "update_request_count"
					metric_type = "gauge"
					unit        = "1"
					interval    = "1m"

					input = observe_datastream.test.dataset

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "initial description"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "update_request_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					description = "updated description"

					metric_name    = "update_request_duration"
					metric_type    = "gauge"
					unit           = "ms"
					interval       = "5m"

					input         = observe_datastream.test.dataset
					shaping_query = "make_col duration:int64(1), service:string(FIELDS)"

					aggregation {
						function = "avg"
						field_path {
							column = "duration"
						}
					}

					metric_tag {
						name   = "service"
						column = "service"
					}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile("aggregation function cannot be changed"),
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDatasetWithTags(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid

					metric_name = "tagged_bytes_total"
					metric_type = "cumulative_counter"
					unit        = "bytes"
					interval    = "1m"

					input         = observe_datastream.test.dataset
					shaping_query = "make_col bytes:int64(1), host:string(FIELDS), region:string(FIELDS)"

					aggregation {
						function = "sum"
						field_path {
							column = "bytes"
						}
					}

					metric_tag {
						name   = "host"
						column = "host"
					}

					metric_tag {
						name   = "region"
						column = "region"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "tagged_bytes_total"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_type", "cumulative_counter"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "sum"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.field_path.0.column", "bytes"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.0.name", "host"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.0.column", "host"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.1.name", "region"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.1.column", "region"),
				),
			},
		},
	})
}

type mockResourceReader struct {
	data map[string]interface{}
}

func (m *mockResourceReader) Get(key string) interface{} {
	v, ok := m.data[key]
	if !ok {
		return nil
	}
	return v
}

func (m *mockResourceReader) GetOk(key string) (interface{}, bool) {
	v, ok := m.data[key]
	return v, ok
}

func TestLogDerivedMetricDefinitionInput_MetricTagsNeverNil(t *testing.T) {
	reader := &mockResourceReader{
		data: map[string]interface{}{
			"metric_name":   "error_count",
			"input":         "o:::dataset:12345",
			"shaping_query": "",
			"aggregation": []interface{}{
				map[string]interface{}{
					"function":   "count",
					"field_path": []interface{}{},
				},
			},
		},
	}

	ldmInput, diags := newLogDerivedMetricDefinitionInput(reader)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if ldmInput.MetricTags == nil {
		t.Fatal("MetricTags must not be nil (would serialize as JSON null)")
	}

	b, err := json.Marshal(ldmInput)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	tags, ok := raw["metricTags"]
	if !ok {
		t.Fatal("metricTags key missing from JSON")
	}
	arr, ok := tags.([]interface{})
	if !ok {
		t.Fatalf("metricTags should be an array, got %T", tags)
	}
	if len(arr) != 0 {
		t.Fatalf("expected empty metricTags array, got length %d", len(arr))
	}
}

func TestLogDerivedMetricDatasetConfig_QueryInputBuilt(t *testing.T) {
	reader := &mockResourceReader{
		data: map[string]interface{}{
			"metric_name":   "req_count",
			"description":   "test",
			"input":         "o:::dataset:12345",
			"shaping_query": "filter true",
			"aggregation": []interface{}{
				map[string]interface{}{
					"function":   "count",
					"field_path": []interface{}{},
				},
			},
		},
	}

	input, queryInput, ldmInput, diags := newLogDerivedMetricDatasetConfig(reader)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if input == nil || queryInput == nil || ldmInput == nil {
		t.Fatal("all outputs must be non-nil")
	}
	if len(queryInput.Stages) != 1 {
		t.Fatalf("expected 1 stage in query, got %d", len(queryInput.Stages))
	}
	if queryInput.OutputStage != ldmDefaultStageID {
		t.Fatalf("expected OutputStage %q, got %q", ldmDefaultStageID, queryInput.OutputStage)
	}
	if queryInput.Stages[0].Id == nil || *queryInput.Stages[0].Id != queryInput.OutputStage {
		t.Fatal("stage ID must match OutputStage")
	}
	if queryInput.Stages[0].Pipeline != "filter true" {
		t.Fatalf("expected pipeline 'filter true', got %q", queryInput.Stages[0].Pipeline)
	}
	if len(queryInput.Stages[0].Input) != 1 || queryInput.Stages[0].Input[0].InputName != ldmDefaultInputName {
		t.Fatalf("expected single input named %q", ldmDefaultInputName)
	}
}

func TestNewLDMShapingStageQueryInput_SingleInput(t *testing.T) {
	reader := &mockResourceReader{
		data: map[string]interface{}{
			"input":         "o:::dataset:12345",
			"shaping_query": "filter severity = \"ERROR\"",
		},
	}

	stageInput, diags := newLDMShapingStageQueryInput(reader)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if stageInput.Id == nil || *stageInput.Id != ldmDefaultStageID {
		t.Fatalf("expected stage id %q, got %#v", ldmDefaultStageID, stageInput.Id)
	}
	if len(stageInput.Input) != 1 {
		t.Fatalf("expected 1 stage input, got %d", len(stageInput.Input))
	}
	if stageInput.Input[0].InputName != ldmDefaultInputName {
		t.Fatalf("expected input name %q, got %q", ldmDefaultInputName, stageInput.Input[0].InputName)
	}
	if *stageInput.Input[0].DatasetId != "12345" {
		t.Fatalf("expected dataset id 12345, got %q", *stageInput.Input[0].DatasetId)
	}
	if stageInput.Pipeline != "filter severity = \"ERROR\"" {
		t.Fatalf("unexpected pipeline: %q", stageInput.Pipeline)
	}
}

func TestLogDerivedMetricDatasetToResourceData_PreservesInputOIDVersion(t *testing.T) {
	const (
		datasetID   = "12345"
		version     = "2026-03-26T12:34:56Z"
		workspaceID = "456"
	)

	data := schema.TestResourceDataRaw(t, resourceLogDerivedMetricDataset().Schema, map[string]interface{}{
		"workspace":     oid.WorkspaceOid(workspaceID).String(),
		"metric_name":   "error_count",
		"input":         oid.OID{Type: oid.TypeDataset, Id: datasetID, Version: stringPtr(version)}.String(),
		"shaping_query": "",
		"aggregation": []interface{}{
			map[string]interface{}{
				"function": "count",
			},
		},
	})

	result := &gql.LogDerivedMetricDataset{
		Id:          "789",
		WorkspaceId: workspaceID,
		Name:        "test-ldm",
		LastSaved:   types.TimeScalar(time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)),
		LogDerivedMetricTable: &gql.LogDerivedMetricDefinition{
			MetricName: "error_count",
			MetricType: gql.MetricTypeGauge,
			Unit:       "1",
			Interval:   types.DurationScalar(time.Minute),
			ShapingQuery: gql.StageQuery{
				Id:       stringPtr(ldmDefaultStageID),
				Pipeline: "",
				Input: []gql.StageQueryInputInputDefinition{
					{
						InputName: ldmDefaultInputName,
						DatasetId: stringPtr(datasetID),
					},
				},
			},
			Aggregation: gql.LogDerivedMetricDefinitionAggregationLogDerivedMetricAggregation{
				Config: &gql.LogDerivedMetricDefinitionAggregationLogDerivedMetricAggregationConfigSimpleLogDerivedMetricAggregationConfig{
					Function: gql.LogDerivedMetricAggregationFunctionCount,
				},
			},
			MetricTags: []gql.LogDerivedMetricDefinitionMetricTagsLogMetricTag{},
		},
	}

	diags := logDerivedMetricDatasetToResourceData(result, data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	expectedOID := oid.OID{Type: oid.TypeDataset, Id: datasetID, Version: stringPtr(version)}.String()
	if got := data.Get("input").(string); got != expectedOID {
		t.Fatalf("expected preserved versioned OID %q, got %q", expectedOID, got)
	}
}

func TestResourceLogDerivedMetricDatasetOptionalDefaultsAreComputed(t *testing.T) {
	schema := resourceLogDerivedMetricDataset().Schema

	for _, field := range []string{"metric_type", "unit", "interval"} {
		if !schema[field].Optional {
			t.Fatalf("%s should be optional", field)
		}
		if !schema[field].Computed {
			t.Fatalf("%s should be computed", field)
		}
	}
}
