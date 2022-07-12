package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Verify we can set default dashboards, read them back, and then delete them
func TestAccObserveDefaultDashboardCreateReadDelete(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	dashboardResource := `
		resource "observe_dashboard" "default_dashboard_testing" {
			workspace = data.observe_workspace.default.oid
			name      = "%s"
			icon_url  = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
				"input": [{
				"inputName": "kubernetes/metrics/Container Metrics",
				"inputRole": "Data",
				"datasetId": "${data.observe_dataset.observation.id}"
				}]
			}]
			EOF
		}`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create a default dashboard
				Config: fmt.Sprintf(configPreamble+"\n"+dashboardResource+`
				resource "observe_default_dashboard" "set_ddb" {
					dataset   = data.observe_dataset.observation.oid
					dashboard = resource.observe_dashboard.default_dashboard_testing.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					// Omit tests for dataset ID because it's difficult to get ahold of one without the version suffix to check against
					resource.TestCheckResourceAttrPair("observe_default_dashboard.set_ddb", "dashboard", "observe_dashboard.default_dashboard_testing", "oid"),
				),
			},
			{
				// Then read it back as a resource
				Config: fmt.Sprintf(configPreamble+"\n"+dashboardResource+`
				resource "observe_default_dashboard" "set_ddb" {
					dataset   = data.observe_dataset.observation.oid
					dashboard = resource.observe_dashboard.default_dashboard_testing.oid
				}

				data "observe_default_dashboard" "read_ddb" {
					dataset = data.observe_dataset.observation.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					// Omit tests for dataset ID because it's difficult to get ahold of one without the version suffix to check against
					resource.TestCheckResourceAttrPair("observe_default_dashboard.set_ddb", "dashboard", "observe_dashboard.default_dashboard_testing", "oid"),
					resource.TestCheckResourceAttrPair("data.observe_default_dashboard.read_ddb", "dashboard", "observe_default_dashboard.set_ddb", "dashboard"),
				),
			},
			{
				// Then clear it
				Config: fmt.Sprintf(configPreamble+"\n"+dashboardResource+`
				data "observe_default_dashboard" "read_ddb" {
					dataset = data.observe_dataset.observation.oid
				}
				`, randomPrefix),
			},
			{
				// And make sure it's gone
				Config: fmt.Sprintf(configPreamble+"\n"+dashboardResource+`
				data "observe_default_dashboard" "read_ddb" {
					dataset = data.observe_dataset.observation.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("data.observe_default_dashboard.read_ddb", "dashboard"),
				),
			},
		},
	})
}
