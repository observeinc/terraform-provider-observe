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
				Config: `
				data "observe_workspace" "kubernetes" {
				  name = "Kubernetes"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_workspace.kubernetes", "id"),
					resource.TestCheckResourceAttrSet("data.observe_workspace.kubernetes", "oid"),
					resource.TestCheckResourceAttr("data.observe_workspace.kubernetes", "name", "Kubernetes"),
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
				data "observe_workspace" "kubernetes" {
				  name = "%s"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(randomPrefix),
			},
		},
	})
}
