package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObservePreferredPathCreate(t *testing.T) {
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
			
				resource "observe_preferred_path" "example" {
					folder  = observe_folder.default.oid
					name    = "%[1]s Path"
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
					resource.TestCheckResourceAttrSet("observe_preferred_path.example", "folder"),
					resource.TestCheckResourceAttrSet("observe_preferred_path.example", "source"),
					resource.TestCheckResourceAttr("observe_preferred_path.example", "description", "Very preferred, much path"),
				),
			},
		},
	})
}
