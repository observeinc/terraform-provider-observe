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
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace_id = data.observe_workspace.default.oid
						rule_kind = "Count"
						name = "%[1]s"
						lookback_time = "30m"
						comment = "a descriptive comment"
						inputs = {
							"test" = observe_datastream.test.dataset
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
						scheduling {
							interval {
								interval = "15m"
								randomize = "0"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace_id"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "Informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.compare_fn", "Greater"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.value_int64", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.interval", "15m"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.randomize", "0"),
				),
			},
		},
	})
}
