package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveMonitorV2ActionEmail(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
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
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
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
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "count"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.interval", "15m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.randomize", "0s"),
				),
			},
		},
	})
}
