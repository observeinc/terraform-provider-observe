package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveBookmarkGroup(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_bookmark_group" "example" {
					workspace 	 = data.observe_workspace.default.oid
					name      	 = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_bookmark_group.example", "workspace"),
					resource.TestCheckResourceAttrSet("observe_bookmark_group.example", "oid"),
					resource.TestCheckResourceAttr("observe_bookmark_group.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_bookmark_group.example", "presentation", "PerCustomerWorkspace"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_bookmark_group" "example" {
					workspace 	 = data.observe_workspace.default.oid
					name      	 = "%[1]s-renamed"
					presentation = "PerUserWorkspace"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_bookmark_group.example", "workspace"),
					resource.TestCheckResourceAttrSet("observe_bookmark_group.example", "oid"),
					resource.TestCheckResourceAttr("observe_bookmark_group.example", "name", randomPrefix+"-renamed"),
					resource.TestCheckResourceAttr("observe_bookmark_group.example", "presentation", "PerUserWorkspace"),
				),
			},
		},
	})
}
