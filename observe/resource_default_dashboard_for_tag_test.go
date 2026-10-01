package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var defaultDashboardForTagDashboard = `
		resource "observe_dashboard" "default_dashboard_for_tag_testing" {
			name      = "%[1]s"
			icon_url  = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/Container Logs",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}`

// Verify we can set default dashboards for tags, read them back, and delete them.
func TestAccObserveDefaultDashboardForTagCreateReadDelete(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	tagName := randomPrefix + "-tag"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create a default dashboard for tag
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.default_dashboard_for_tag_testing.oid
				}
				`, randomPrefix, tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_dashboard_for_tag.set_ddb", "tag", tagName),
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.default_dashboard_for_tag_testing", "oid"),
				),
			},
			{
				ResourceName:      "observe_default_dashboard_for_tag.set_ddb",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Then read it back as a data source
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.default_dashboard_for_tag_testing.oid
				}

				data "observe_default_dashboard_for_tag" "read_ddb" {
					tag = "%[2]s"
				}
				`, randomPrefix, tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_dashboard_for_tag.set_ddb", "tag", tagName),
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.default_dashboard_for_tag_testing", "oid"),
					resource.TestCheckResourceAttrPair("data.observe_default_dashboard_for_tag.read_ddb", "dashboard", "observe_default_dashboard_for_tag.set_ddb", "dashboard"),
					resource.TestCheckResourceAttr("data.observe_default_dashboard_for_tag.read_ddb", "tag", tagName),
				),
			},
			{
				// Then clear it
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				data "observe_default_dashboard_for_tag" "read_ddb" {
					tag = "%[2]s"
				}
				`, randomPrefix, tagName),
			},
			{
				// And make sure it's gone
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				data "observe_default_dashboard_for_tag" "read_ddb" {
					tag = "%[2]s"
				}
				`, randomPrefix, tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_default_dashboard_for_tag.read_ddb", "dashboard", ""),
				),
			},
		},
	})
}

func TestAccObserveDefaultDashboardForTagUpdateDashboard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	tagName := randomPrefix + "-tag"

	dashboards := `
		resource "observe_dashboard" "first" {
			name      = "%[1]s-first"
			icon_url  = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/Container Logs",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}

		resource "observe_dashboard" "second" {
			name      = "%[1]s-second"
			icon_url  = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/Container Logs",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(dashboards+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.first.oid
				}
				`, randomPrefix, tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.first", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(dashboards+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.second.oid
				}
				`, randomPrefix, tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.second", "oid"),
				),
			},
		},
	})
}

// Changing tag ForceNews the resource: the old tag binding is cleared and the
// new tag is bound to the same dashboard.
func TestAccObserveDefaultDashboardForTagForceNewTag(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	tagOld := randomPrefix + "-old"
	tagNew := randomPrefix + "-new"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.default_dashboard_for_tag_testing.oid
				}
				`, randomPrefix, tagOld),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_dashboard_for_tag.set_ddb", "tag", tagOld),
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.default_dashboard_for_tag_testing", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(defaultDashboardForTagDashboard+`
				resource "observe_default_dashboard_for_tag" "set_ddb" {
					tag       = "%[2]s"
					dashboard = resource.observe_dashboard.default_dashboard_for_tag_testing.oid
				}

				data "observe_default_dashboard_for_tag" "old_tag" {
					tag        = "%[3]s"
					depends_on = [observe_default_dashboard_for_tag.set_ddb]
				}

				data "observe_default_dashboard_for_tag" "new_tag" {
					tag        = "%[2]s"
					depends_on = [observe_default_dashboard_for_tag.set_ddb]
				}
				`, randomPrefix, tagNew, tagOld),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_dashboard_for_tag.set_ddb", "tag", tagNew),
					resource.TestCheckResourceAttrPair("observe_default_dashboard_for_tag.set_ddb", "dashboard", "observe_dashboard.default_dashboard_for_tag_testing", "oid"),
					resource.TestCheckResourceAttrPair("data.observe_default_dashboard_for_tag.new_tag", "dashboard", "observe_dashboard.default_dashboard_for_tag_testing", "oid"),
					resource.TestCheckResourceAttr("data.observe_default_dashboard_for_tag.old_tag", "dashboard", ""),
				),
			},
		},
	})
}
