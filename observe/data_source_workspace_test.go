package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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
					resource.TestCheckResourceAttr("data.observe_workspace.kubernetes", "name", "Kubernetes"),
				),
			},
		},
	})
}
