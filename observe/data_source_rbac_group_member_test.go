package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceRbacGroupmember(t *testing.T) {
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
					name = "%[2]s"
				}

				resource "observe_rbac_group_member" "example" {
					group = observe_rbac_group.example.oid
					member {
						user = data.observe_user.system.oid
					}
				}

				data "observe_rbac_group_member" "lookup" {
					id = observe_rbac_group_member.example.id
				}
				`, systemUser(), randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_rbac_group_member.lookup", "id"),
					resource.TestCheckResourceAttrSet("data.observe_rbac_group_member.lookup", "oid"),
					resource.TestCheckResourceAttrPair("data.observe_rbac_group_member.lookup", "group", "observe_rbac_group.example", "oid"),
					resource.TestCheckResourceAttr("data.observe_rbac_group_member.lookup", "member.#", "1"),
					resource.TestCheckResourceAttrPair("data.observe_rbac_group_member.lookup", "member.0.user", "data.observe_user.system", "oid"),
				),
			},
		},
	})
}
