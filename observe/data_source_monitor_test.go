package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceMonitor(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						disabled  = true

						description = "description"
						comment     = "comment"
						is_template = true
						definition = jsonencode({ "hello" = "world" })

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						stage {}

						rule {
							count {
								compare_function   = "less_or_equal"
								compare_values     = [1]
								lookback_time      = "1m"
							}
						}

						notification_spec {
							importance      = "informational"
							notify_on_close = true
						}
					}

					data "observe_monitor" "lookup" {
						id         = observe_monitor.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "disabled", "true"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "is_template", "true"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "description", "description"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "definition", `{"hello":"world"}`),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "comment", "comment"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						description = "description"
						comment     = "comment"

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}

						rule {
							count {
								compare_function   = "less_or_equal"
								compare_values     = [1]
								lookback_time      = "1m"
							}
						}
					}

					data "observe_monitor" "lookup" {
						id         = observe_monitor.first.id
					}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", "filter false\n"),
				),
			},
		},
	})
}

func TestAccObserveSourceMonitorLookup(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						stage {}

						rule {
							count {
								compare_function   = "less_or_equal"
								compare_values     = [1]
								lookback_time      = "1m"
							}
						}
					}

					data "observe_monitor" "lookup" {
						workspace = data.observe_workspace.default.oid
						name      = observe_monitor.first.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						stage {}

						rule {
							source_column = "OBSERVATION_INDEX"

							threshold {
								compare_function = "greater"
								compare_values = [ 75, ]
								lookback_time = "5m0s"
							}
						}

						notification_spec {
							importance      = "informational"
						}
					}

					data "observe_monitor" "lookup" {
						workspace = data.observe_workspace.default.oid
						name      = observe_monitor.first.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.threshold.0.compare_function", "greater"),
				),
			},
		},
	})
}

func TestAccObserveSourceMonitorLog(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
						make_col vt:BUNDLE_TIMESTAMP
						make_interval vt
						EOF
					}
				}

				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_dataset.first.oid
					}

					stage {
						pipeline = <<-EOF
							filter OBSERVATION_INDEX != 0
						EOF
					}
					stage {
						pipeline = "timechart 1m, frame(back:10m), A_ContainerLogsClean_count:count(), group_by()"
					}

					rule {
						source_column = "A_ContainerLogsClean_count"

						log {
							compare_function   = "greater"
							compare_values     = [1]
							lookback_time      = "1m"
							expression_summary = "Some text"
							source_log_dataset = observe_dataset.first.oid
							log_stage_id = "stage-0"
						}
					}

					notification_spec {
						merge      = "separate"
					}
				}

				data "observe_monitor" "lookup" {
					id         = observe_monitor.first.id
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.expression_summary", "Some text"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.log_stage_id", "stage-0"),
				),
			},
		},
	})
}
