package observe

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveLogDerivedMetricDatasetCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s"
					description = "test log-derived metric"

					metric_name = "error_count"
					metric_type = "gauge"
					unit        = "1"
					interval    = "1m"

					shaping_query {
						inputs = {
							"logs" = observe_datastream.test.dataset
						}
						pipeline = ""
					}

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "workspace"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "test log-derived metric"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "error_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_type", "gauge"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "unit", "1"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "oid"),
				),
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
					name        = "%[1]s"
					description = "initial description"

					metric_name = "request_count"
					metric_type = "gauge"
					unit        = "1"
					interval    = "1m"

					shaping_query {
						inputs = {
							"logs" = observe_datastream.test.dataset
						}
						pipeline = ""
					}

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "initial description"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "request_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-updated"
					description = "updated description"

					metric_name = "request_duration"
					metric_type = "gauge"
					unit        = "ms"
					interval    = "5m"

					shaping_query {
						inputs = {
							"logs" = observe_datastream.test.dataset
						}
						pipeline = <<-EOF
							make_col duration:int64(duration_ms)
						EOF
					}

					aggregation {
						function = "avg"
						field_path {
							column = "duration"
							path   = ""
						}
					}

					metric_tag {
						name   = "service"
						column = "service"
						path   = ""
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "updated description"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "request_duration"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "unit", "ms"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "avg"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.field_path.0.column", "duration"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.0.name", "service"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tag.0.column", "service"),
				),
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
					name      = "%[1]s"

					metric_name = "bytes_total"
					metric_type = "cumulative_counter"
					unit        = "bytes"
					interval    = "1m"

					shaping_query {
						inputs = {
							"logs" = observe_datastream.test.dataset
						}
						pipeline = ""
					}

					aggregation {
						function = "sum"
						field_path {
							column = "bytes"
							path   = ""
						}
					}

					metric_tag {
						name   = "host"
						column = "host"
						path   = ""
					}

					metric_tag {
						name   = "region"
						column = "region"
						path   = ""
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "bytes_total"),
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
			"metric_name": "error_count",
			"shaping_query": []interface{}{
				map[string]interface{}{
					"inputs": map[string]interface{}{
						"logs": "o:::dataset:12345",
					},
					"pipeline": "",
					"stage_id": "",
				},
			},
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
			"name":        "test-ldm",
			"metric_name": "req_count",
			"description": "test",
			"shaping_query": []interface{}{
				map[string]interface{}{
					"inputs": map[string]interface{}{
						"logs": "o:::dataset:12345",
					},
					"pipeline": "filter true",
					"stage_id": "",
				},
			},
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
	if queryInput.OutputStage == "" {
		t.Fatal("OutputStage must be set")
	}
	if queryInput.Stages[0].Id == nil || *queryInput.Stages[0].Id != queryInput.OutputStage {
		t.Fatal("stage ID must match OutputStage")
	}
}
