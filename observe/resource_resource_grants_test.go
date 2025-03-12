package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveResourceGrantsDataset(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-1"
					inputs = {
						"test" = observe_datastream.test.dataset
					}
					stage {}
				}

				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				data "observe_rbac_group" "everyone" {
					name = "Everyone"
				}

				resource "observe_resource_grants" "test" {
					oid = observe_dataset.test.oid

					grant {
						subject = observe_rbac_group.example.oid
						role    = "dataset_editor"
					}
					grant {
						subject = data.observe_rbac_group.everyone.oid
						role    = "dataset_viewer"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_resource_grants.test", "oid"),
					resource.TestCheckResourceAttr("observe_resource_grants.test", "grant.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_editor"}),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_viewer"}),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-1"
					inputs = {
						"test" = observe_datastream.test.dataset
					}
					stage {}
				}

				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				data "observe_rbac_group" "everyone" {
					name = "Everyone"
				}

				resource "observe_resource_grants" "test" {
					oid = observe_dataset.test.oid

					grant {
						subject = observe_rbac_group.example.oid
						role    = "dataset_editor"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_resource_grants.test", "oid"),
					resource.TestCheckResourceAttr("observe_resource_grants.test", "grant.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_editor"}),
				),
			},
		},
	})
}
