package observe

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Verify we can create workspace
// Only running this test in CI.
// Requires the additional_workspaces customer config to be set to allow
func TestAccObserveWorkspaceCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("CI") != "true" {
				t.Skip("Skipping test: CI environment variable is not set to 'true'")
			}
		},
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
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace.first", "name", randomPrefix+"-renamed"),
				),
			},
		},
	})
}
