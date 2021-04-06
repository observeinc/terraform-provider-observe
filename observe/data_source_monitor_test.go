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
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.kubernetes.oid
						name      = "%s"

						inputs = {
							"observation" = data.observe_dataset.observation.oid
						}

						stage {
							pipeline = "filter true"
						}

						rule {
							source_column = "OBSERVATION_KIND"
							group_by      = "none"

							count {
								compare_function   = "between_half_open"
								compare_values     = [1, 10]
								lookback_time      = "1m"
							}
						}

						notification_spec {
							selection       = "count"
							selection_value = 1
						}
					}

					data "observe_monitor" "lookup" {
						workspace  = data.observe_workspace.kubernetes.oid
						id         = observe_monitor.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", "filter true"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "notification_spec.0.selection", "count"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "notification_spec.0.selection_value", "1"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.kubernetes.oid
						name      = "%s"

						inputs = {
							"observation" = data.observe_dataset.observation.oid
						}

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}

						rule {
							source_column = "OBSERVATION_KIND"
							group_by      = "none"

							count {
								compare_function   = "between_half_open"
								compare_values     = [1, 10]
								lookback_time      = "1m"
							}
						}

						notification_spec {
							selection       = "count"
							selection_value = 1
						}
					}

					data "observe_monitor" "lookup" {
						workspace  = data.observe_workspace.kubernetes.oid
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
