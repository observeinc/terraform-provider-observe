package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatastreamDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_datastream" "a" {
			name = "%[1]s"
		}

		data "observe_datastream" "lookup" {
			name = observe_datastream.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_datastream.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveDatasetDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_dataset" "b" {
			name = "%[1]s-b"
			inputs = { "a" = observe_datastream.test_no_ws.dataset }
			stage { pipeline = "filter false" }
		}

		data "observe_dataset" "lookup" {
			name = observe_dataset.b.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_dataset.lookup", "name", randomPrefix+"-b"),
		),
	})
}

func TestAccObserveFolderDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_folder" "a" {
			name = "%[1]s"
		}

		data "observe_folder" "lookup" {
			name = observe_folder.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_folder.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveWorksheetDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_worksheet" "a" {
			name = "%[1]s"
			queries = <<-EOF
			[{
				"id": "stage",
				"pipeline": "filter field = \"cpu_usage_core_seconds\"",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}

		data "observe_worksheet" "lookup" {
			id = observe_worksheet.a.id
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_worksheet.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_monitor" "a" {
			name = "%[1]s"
			inputs = { "test" = observe_datastream.test_no_ws.dataset }
			stage {}
			rule {
				count {
					compare_function = "less_or_equal"
					compare_values   = [1]
					lookback_time    = "1m"
				}
			}
		}

		data "observe_monitor" "lookup" {
			name = observe_monitor.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorV2DataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_monitor_v2" "a" {
			rule_kind     = "count"
			name          = "%[1]s"
			lookback_time = "30m"
			inputs = { "test" = observe_datastream.test_no_ws.dataset }
			stage {
				pipeline     = "colmake kind:\"test\""
				output_stage = true
			}
			stage { pipeline = "filter kind ~ \"test\"" }
			rules {
				level = "informational"
				count {
					compare_values {
						compare_fn  = "greater"
						value_int64 = [0]
					}
				}
			}
			scheduling {
				transform { freshness_goal = "15m" }
			}
		}

		data "observe_monitor_v2" "lookup" {
			name = observe_monitor_v2.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorActionDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_monitor_action" "a" {
			name = "%[1]s"
			webhook {
				url_template  = "https://example.com"
				body_template = "{}"
			}
		}

		data "observe_monitor_action" "lookup" {
			name = observe_monitor_action.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorV2ActionDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_monitor_v2_action" "a" {
			type = "webhook"
			name = "%[1]s"
			webhook {
				url    = "https://example.com/"
				method = "post"
				body   = "body"
			}
		}

		data "observe_monitor_v2_action" "lookup" {
			name = observe_monitor_v2_action.a.name
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_monitor_v2_action.lookup", "name", randomPrefix),
		),
	})
}

func TestAccObserveDashboardDataSourceNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_dashboard" "a" {
			name     = "%[1]s"
			icon_url = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}

		data "observe_dashboard" "lookup" {
			id = observe_dashboard.a.id
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
		),
	})
}
