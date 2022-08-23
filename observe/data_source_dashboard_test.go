package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDashboard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_dashboard" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%s"
						icon_url  = "test"
						stages = <<-EOF
						[{
							"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
							"input": [{
							  "inputName": "kubernetes/metrics/Container Metrics",
							  "inputRole": "Data",
							  "datasetId": "41042989"
							}]
						}]
						EOF
					}

					data "observe_dashboard" "lookup" {
						id        = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
				),
			},
		},
	})
}
