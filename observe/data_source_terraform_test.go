package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDatasetTerraform(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_datastream" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s-a"
					}

					resource "observe_dataset" "b" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s-b"

						inputs = { "a" = observe_datastream.a.dataset }

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}
					}

					data "observe_terraform" "lookup_by_name" {
						target = observe_dataset.b.oid
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
