package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// todo write some tests
// starting point: count, threshold, ????????????

var monitorV2ConfigPreamble = configPreamble + datastreamConfigPreamble

func TestAccObserveMonitorV2(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
					data "observe_dataset" "test" {
						workspace = data.observe_workspace.default.oid
						name      = observe_datastream.test.name
					}

					resource "observe_monitor_v2" "first" {
						workspace_id = data.observe_workspace.default.oid
						rule_kind = "Count"
						name = "%[1]s"
						lookback_time = "30m"
						comment = "a descriptive comment"
						inputs = {
							"battery" = data.observe_dataset.default.oid
						}
						stage {
							pipeline = <<-EOF
								colmake kind:"test", description:"test"
							EOF
							output_stage = true
						}
						stage {
							pipeline = <<-EOF
								filter kind ~ "test"
							EOF
						}
						rules {
							level = "Informational"
							count {
								compare_values {
									compare_fn = "Greater"
									value_int64 = [0]
								}
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace_id"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "comment", "a descriptive comment"),
				),
			},
		},
	})
}
