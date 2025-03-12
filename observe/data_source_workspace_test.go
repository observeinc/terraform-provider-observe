package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceWorkspace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_workspace" "default" {
					name = "%s"
				}


				data "observe_workspace" "default_by_id" {
					id = data.observe_workspace.default.id
				}

				`, defaultWorkspaceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_workspace.default", "id"),
					resource.TestCheckResourceAttrSet("data.observe_workspace.default", "oid"),
					resource.TestCheckResourceAttr("data.observe_workspace.default", "name", defaultWorkspaceName),
					resource.TestCheckResourceAttr("data.observe_workspace.default_by_id", "name", defaultWorkspaceName),
				),
			},
		},
	})
}

func TestAccObserveSourceWorkspaceNotFound(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_workspace" "default" {
					name = "%s"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(randomPrefix),
			},
		},
	})
}
