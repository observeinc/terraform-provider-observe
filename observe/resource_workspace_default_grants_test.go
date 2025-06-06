package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveWorkspaceDefaultGrants(t *testing.T) {
	t.Skip()
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

				resource "observe_workspace_default_grants" "test" {
				  group {
					oid        = observe_rbac_group.engineering.oid
					permission = "edit"
					object_types = ["dashboard", "worksheet"]
				  }

				  group {
					oid        = observe_rbac_group.engineering.oid
					permission = "view"
					object_types = ["datastream"]
				  }

				  group {
					oid        = observe_rbac_group.readonly.oid
					permission = "view"
				  }
				}
				`, randomPrefix1, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.#", "3"),
					resource.TestCheckResourceAttrSet("observe_workspace_default_grants.test", "group.0.oid"),
					resource.TestCheckResourceAttrSet("observe_workspace_default_grants.test", "group.1.oid"),
					resource.TestCheckResourceAttrSet("observe_workspace_default_grants.test", "group.2.oid"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.permission", "edit"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.1.permission", "view"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.2.permission", "view"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.object_types.#", "2"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.object_types.0", "dashboard"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.object_types.1", "worksheet"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.1.object_types.#", "1"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.1.object_types.0", "datastream"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.2.object_types.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_rbac_group" "everyone" {
				  name = "Everyone"
				}

				resource "observe_workspace_default_grants" "test" {
				  group {
					oid        = data.observe_rbac_group.everyone.oid
					permission = "edit"
				  }
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.#", "1"),
					resource.TestCheckResourceAttrSet("observe_workspace_default_grants.test", "group.0.oid"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.permission", "edit"),
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.0.object_types.#", "0"),
				),
			},
		},
	})
}

func TestAccObserveWorkspaceDefaultGrantsEmpty(t *testing.T) {
	t.Skip()
	// need to be able to set "only creator gets edit access by default"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				resource "observe_workspace_default_grants" "test" {
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_workspace_default_grants.test", "group.#", "0"),
				),
			},
		},
	})
}
