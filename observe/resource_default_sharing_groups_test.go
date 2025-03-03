package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDefaultSharingGroups(t *testing.T) {
	randomPrefix1 := acctest.RandomWithPrefix("tf")
	randomPrefix2 := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "engineering" {
				  name = "%[1]s"
				}

				resource "observe_rbac_group" "readonly" {
				  name = "%[2]s"
				}

				resource "observe_default_sharing_groups" "test" {
				  group {
					oid        = observe_rbac_group.engineering.oid
					permission = "edit"
				  }

				  group {
					oid        = observe_rbac_group.readonly.oid
					permission = "view"
				  }
				}
				`, randomPrefix1, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.#", "2"),
					resource.TestCheckResourceAttrSet("observe_default_sharing_groups.test", "group.0.oid"),
					resource.TestCheckResourceAttrSet("observe_default_sharing_groups.test", "group.1.oid"),
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.0.permission", "edit"),
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.1.permission", "view"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_rbac_group" "everyone" {
				  name = "Everyone"
				}

				resource "observe_default_sharing_groups" "test" {
				  group {
					oid        = data.observe_rbac_group.everyone.oid
					permission = "edit"
				  }
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.#", "1"),
					resource.TestCheckResourceAttrSet("observe_default_sharing_groups.test", "group.0.oid"),
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.0.permission", "edit"),
				),
			},
		},
	})
}

func TestAccObserveDefaultSharingGroupsEmpty(t *testing.T) {
	// need to be able to set "only creator gets edit access by default"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				resource "observe_default_sharing_groups" "test" {
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_default_sharing_groups.test", "group.#", "0"),
				),
			},
		},
	})
}
