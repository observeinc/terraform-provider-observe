package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceFolder(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_folder" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
					}

					data "observe_folder" "lookup_by_name" {
						workspace  = data.observe_workspace.default.oid
						name       = "%[1]s"
						depends_on = [observe_folder.a]
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_folder.lookup_by_name", "name", randomPrefix),
				),
			},
		},
	})
}
