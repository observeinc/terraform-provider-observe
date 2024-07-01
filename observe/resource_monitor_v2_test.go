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
					resource "observe_monitor_v2" "first" {
						workspace_id = data.observe_workspace.default.oid
						rule_kind = "Count"
						name = "owen's special monitorv2"
						lookback_time = "30m"
						inputs = {
							"battery" = data.observe_dataset.default.oid
						}
						stage {
							pipeline = <<-EOF
								filter DATASTREAM_ID = "4f7fc854-53ae-4ace-8530-906417001"
							EOF
							output_stage = true
						}
						rules {
							level = "Informational"
							count {
								compare_values {
									compare_fn = "Greater"
									value_int64 = 0
								}
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "freshness", "4m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("observe_monitor.first", "definition", `{"hello":"world"}`),
				),
			},
		},
	})
}
