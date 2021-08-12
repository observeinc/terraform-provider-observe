package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Verify we can create workspace
func TestAccObserveWorkspaceCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_workspace" "first" {
					name 	  = "%s"
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace.first", "name", randomPrefix),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_workspace" "first" {
					name 	  = "%s-renamed"
				}

				resource "observe_dataset" "first" {
					workspace = observe_workspace.first.oid
					name      = "Test"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace.first", "name", randomPrefix+"-renamed"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "Test"),
				),
			},
		},
	})
}
