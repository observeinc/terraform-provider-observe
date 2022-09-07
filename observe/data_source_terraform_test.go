package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDatasetTerraform(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_dataset" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = { "observation" = data.observe_dataset.observation.oid }

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}
					}

					data "observe_terraform" "lookup_by_name" {
						target = observe_dataset.a.oid
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_terraform.lookup_by_name", "resource"),
					resource.TestCheckResourceAttrSet("data.observe_terraform.lookup_by_name", "data_source"),
				),
			},
		},
	})
}
