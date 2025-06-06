package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDashboardLinkCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := "tf" + acctest.RandString(20)
	t.Log("random prefix=", randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_folder" "default" {
					workspace  = data.observe_workspace.default.oid
					name       = "%[1]s"
				}

				resource "observe_dashboard" "a_to_b" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s a"
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

				resource "observe_dashboard" "b_to_a" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s b"
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

				resource "observe_dashboard_link" "example_with_folder" {
					folder         = observe_folder.default.oid
					name           = "%[1]s Link (Folder)"
					description    = "Very linked, much dashboard"
					from_dashboard = observe_dashboard.a_to_b.oid
					to_dashboard   = observe_dashboard.b_to_a.oid
					from_card      = "some card"
					link_label     = "go hither"
				}

				resource "observe_dashboard_link" "example_with_workspace" {
					workspace      = data.observe_workspace.default.oid
					name           = "%[1]s Link (Workspace)"
					description    = "Very linked, much dashboard"
					from_dashboard = observe_dashboard.b_to_a.oid
					to_dashboard   = observe_dashboard.a_to_b.oid
					link_label     = "go yon"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dashboard_link.example_with_folder", "folder"),
					resource.TestCheckResourceAttrSet("observe_dashboard_link.example_with_folder", "workspace"),
					resource.TestCheckResourceAttr("observe_dashboard_link.example_with_folder", "description", "Very linked, much dashboard"),
					resource.TestCheckResourceAttr("observe_dashboard_link.example_with_folder", "link_label", "go hither"),

					resource.TestCheckResourceAttrSet("observe_dashboard_link.example_with_workspace", "folder"),
					resource.TestCheckResourceAttrSet("observe_dashboard_link.example_with_workspace", "workspace"),
					resource.TestCheckResourceAttr("observe_dashboard_link.example_with_workspace", "description", "Very linked, much dashboard"),
					resource.TestCheckResourceAttr("observe_dashboard_link.example_with_workspace", "link_label", "go yon"),
				),
			},
		},
	})
}

func TestAccDashboardLinkWithoutFolderOrWorkspace(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := "tf" + acctest.RandString(20)
	t.Log("random prefix=", randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_folder" "default" {
					workspace  = data.observe_workspace.default.oid
					name       = "%[1]s"
				}

				resource "observe_dashboard" "a_to_b" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s a"
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

				resource "observe_dashboard" "b_to_a" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s b"
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

				resource "observe_dashboard_link" "example_without_folder_or_workspace" {
					name           = "%[1]s Link (Folder)"
					description    = "Very linked, much dashboard"
					from_dashboard = observe_dashboard.a_to_b.oid
					to_dashboard   = observe_dashboard.b_to_a.oid
					link_label     = "going nowhere"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(`one of .folder,workspace. must be specified`),
			},
		},
	})

}
