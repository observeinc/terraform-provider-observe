package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObservePreferredPathCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "a_to_b" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "to_b"
				}

				resource "observe_link" "b_to_a" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.b.oid
					target    = observe_dataset.a.oid
					fields    = ["key:key"]
					label     = "to_a"
				}

			
				resource "observe_folder" "default" {
					workspace  = data.observe_workspace.default.oid
					name       = "%[1]s"
				}
			
				resource "observe_preferred_path" "example_with_folder" {
					folder      = observe_folder.default.oid
					name        = "%[1]s Path (Folder)"
					description = "Very preferred, much path"

					source    = observe_dataset.a.oid

					step {
						link = observe_link.a_to_b.oid
					}

					step {
						link = observe_link.b_to_a.oid
					}
				}

				resource "observe_preferred_path" "example_with_workspace" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s Path (Workspace)"
					description = "Very preferred, much path"

					source    = observe_dataset.a.oid

					step {
						link = observe_link.a_to_b.oid
					}

					step {
						link = observe_link.b_to_a.oid
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_preferred_path.example_with_folder", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.example_with_folder", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.example_with_folder", "description", "Very preferred, much path"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.example_with_workspace", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.example_with_workspace", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.example_with_workspace", "description", "Very preferred, much path"),
				),
			},
		},
	})
}

func TestAccObservePreferredPathUpdate_Reverse(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// create a link: a -> b
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "this" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "update-reverse"
				}

				resource "observe_preferred_path" "this" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s Path (Workspace)"
					description = "Very preferred, much path"

					source    = observe_dataset.a.oid

					step {
						link = observe_link.this.oid
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.this", "description", "Very preferred, much path"),
				),
			},
			{
				// reverse the link: b -> a
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "this" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.b.oid
					target    = observe_dataset.a.oid
					fields    = ["key:key"]
					label     = "to_b"
				}

				resource "observe_preferred_path" "this" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s Path (Workspace)"
					description = "Very preferred, much path"

					source    = observe_dataset.a.oid

					step {
						link = observe_link.this.oid
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.this", "description", "Very preferred, much path"),
				),
			},
			{
				// update the preferred path step (reverse)
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "this" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.b.oid
					target    = observe_dataset.a.oid
					fields    = ["key:key"]
					label     = "to_b"
				}

				resource "observe_preferred_path" "this" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s Path (Workspace)"
					description = "Very preferred, much path"

					source    = observe_dataset.a.oid

					step {
						link = observe_link.this.oid
						reverse = true
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.this", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.this", "description", "Very preferred, much path"),
					resource.TestCheckResourceAttr("observe_preferred_path.this", "step.0.reverse", "true"),
				),
			},
		},
	})
}
