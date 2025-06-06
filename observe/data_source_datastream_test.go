package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDatastream(t *testing.T) {
	t.Skip()
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
						name      = "%[1]s"
					}

					data "observe_datastream" "lookup_by_name" {
						workspace = data.observe_workspace.default.oid
						name      = observe_datastream.a.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_datastream.lookup_by_name", "name", randomPrefix),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
						resource "observe_datastream" "a" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s"
						}

						data "observe_datastream" "lookup_by_id" {
							workspace = data.observe_workspace.default.oid
							id        = observe_datastream.a.id
						}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_datastream.lookup_by_id", "name", randomPrefix),
				),
			},
		},
	})
}
