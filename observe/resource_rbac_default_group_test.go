package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveRbacDefaultGroupSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_rbac_group" "writer" {
					name      = "writer"
				}

				resource "observe_rbac_default_group" "writer" {
					group = data.observe_rbac_group.writer.oid
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_rbac_default_group.writer", "group"),
				),
			},
		},
	})
}
