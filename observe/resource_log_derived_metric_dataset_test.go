package observe

import (
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
