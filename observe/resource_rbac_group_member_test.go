package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveRbacGroupmemberWithUserCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				data "observe_user" "system" {
                  email = "%[1]s"
                }

				resource "observe_rbac_group" "example" {
				  name      = "%[2]s"
				}

				resource "observe_rbac_group_member" "example" {
				  group = observe_rbac_group.example.oid
				  member {
				    user= data.observe_user.system.oid
				  }
				}
				`, systemUser(), randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_rbac_group_member.example", "group"),
					resource.TestCheckResourceAttr("observe_rbac_group_member.example", "member.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_group_member.example", "member.0.user"),
				),
			},
		},
	})
}
