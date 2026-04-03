package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveLogDerivedMetricDataset_Basic(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric"

					metric_name = "error_count"
					metric_type = "gauge"
					unit        = "errors"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "workspace"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix+"-log-metric"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "error_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_type", "gauge"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "unit", "errors"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "shaping_query", "filter true"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
					resource.TestCheckResourceAttrSet("observe_log_derived_metric_dataset.test", "oid"),
				),
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDataset_Update(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric"

					metric_name = "request_count"
					metric_type = "counter"
					unit        = "requests"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}

					metric_tags {
						name         = "status"
						field_column = "status_code"
						field_path   = "."
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "request_count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_type", "counter"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "count"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tags.0.name", "status"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tags.0.field_column", "status_code"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric-updated"
					description = "Updated log derived metric"

					metric_name = "request_count"
					metric_type = "counter"
					unit        = "requests"
					interval    = "5m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter severity = \"INFO\""

					aggregation {
						function = "count"
					}

					metric_tags {
						name         = "status"
						field_column = "status_code"
						field_path   = "."
					}

					metric_tags {
						name         = "method"
						field_column = "http_method"
						field_path   = "."
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "name", randomPrefix+"-log-metric-updated"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "Updated log derived metric"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "interval", "5m0s"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "shaping_query", "filter severity = \"INFO\""),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tags.#", "2"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_tags.1.name", "method"),
				),
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDataset_WithFieldAggregation(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric-sum"

					metric_name = "total_bytes"
					metric_type = "gauge"
					unit        = "bytes"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function     = "sum"
						field_column = "bytes_sent"
						field_path   = "."
					}

					metric_tags {
						name         = "service"
						field_column = "service_name"
						field_path   = "."
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "metric_name", "total_bytes"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.function", "sum"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.field_column", "bytes_sent"),
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "aggregation.0.field_path", "."),
				),
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDataset_Import(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric-import"

					metric_name = "import_test"
					metric_type = "gauge"
					unit        = "count"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
			},
			{
				ResourceName:      "observe_log_derived_metric_dataset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccObserveLogDerivedMetricDataset_Description(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-log-metric"
					description = "Test log derived metric"

					metric_name = "test_metric"
					metric_type = "gauge"
					unit        = "count"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "Test log derived metric"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-log-metric"
					description = "Updated description"

					metric_name = "test_metric"
					metric_type = "gauge"
					unit        = "count"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", "Updated description"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_log_derived_metric_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-log-metric"

					metric_name = "test_metric"
					metric_type = "gauge"
					unit        = "count"
					interval    = "1m"

					input_dataset = observe_datastream.test.dataset
					shaping_query = "filter true"

					aggregation {
						function = "count"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_log_derived_metric_dataset.test", "description", ""),
				),
			},
		},
	})
}
