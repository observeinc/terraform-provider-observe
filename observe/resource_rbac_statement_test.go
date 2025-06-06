package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveRbacStatementWithGroupCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skipf("skipping, cannot create rbac v1 statements after rbac v2 is enabled, to be removed")
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						group = observe_rbac_group.example.oid
					}
					object {
						workspace = data.observe_workspace.default.id
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.group"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.workspace"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				resource "observe_folder" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
				}

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						group = observe_rbac_group.example.oid
					}
					object {
						id = observe_folder.example.id
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.group"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.id"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				resource "observe_folder" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
				}

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						group = observe_rbac_group.example.oid
					}
					object {
						folder = observe_folder.example.id
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.group"),
				),
			},
		},
	})
}

func TestAccObserveRbacStatementWithUserCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skipf("skipping, cannot create rbac v1 statements after rbac v2 is enabled, to be removed")
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

				resource "observe_rbac_statement" "example" {
					description = "%[2]s"
					subject {
						user = data.observe_user.system.oid
					}
					object {
						workspace = data.observe_workspace.default.id
					}
					role = "Lister"
				}
				`, systemUser(), randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.user"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.workspace"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				data "observe_user" "system" {
                  email = "%[1]s"
                }

				resource "observe_folder" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[2]s"
				}

				resource "observe_rbac_statement" "example" {
					description = "%[2]s"
					subject {
						user = data.observe_user.system.oid
					}
					object {
						id = observe_folder.example.id
					}
					role = "Lister"
				}
				`, systemUser(), randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.user"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.id"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
		},
	})
}

func TestAccObserveRbacStatementAllCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skipf("skipping, cannot create rbac v1 statements after rbac v2 is enabled, to be removed")
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						all = true
					}
					object {
						all = true
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.all"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.all"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
		},
	})
}

func TestAccObserveRbacStatementTypeCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skipf("skipping, cannot create rbac v1 statements after rbac v2 is enabled, to be removed")
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						all = true
					}
					object {
						type = "dataset"
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.all"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.type"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						all = true
					}
					object {
						type = "dataset"
						name = "test"
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.all"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.type"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.name"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`

				resource "observe_rbac_statement" "example" {
					description = "%[1]s"
					subject {
						all = true
					}
					object {
						type = "dataset"
					owner = true
					}
					role = "Lister"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "description", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "subject.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "subject.0.all"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "object.#", "1"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.type"),
					resource.TestCheckResourceAttrSet("observe_rbac_statement.example", "object.0.owner"),
					resource.TestCheckResourceAttr("observe_rbac_statement.example", "role", "Lister"),
				),
			},
		},
	})
}
