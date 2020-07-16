package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceForeignKey(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// adding foreign key will increment version of affected datasets
				ExpectNonEmptyPlan: true,

				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}

				data "observe_fk" "check" {
					source     = observe_dataset.a.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
					// wait for foreign key to be set
					depends_on = [observe_fk.example]
				}
				`, randomPrefix),
			},
		},
	})
}
func TestAccObserveSourceForeignKeyErrors(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(fkConfigPreamble+`
					data "observe_fk" "check" {
						source     = observe_dataset.a.oid
						target     = observe_dataset.b.oid
						fields     = ["missing"]
					}`, randomPrefix),
				ExpectError: regexp.MustCompile("not found"),
			},
		},
	})
}
