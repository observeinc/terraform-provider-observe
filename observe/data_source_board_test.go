package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceBoard(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_board" "first" {
					dataset  = observe_datastream.test.dataset
					name     = "%[1]s"
					type     = "set"
					json     = "{}"
				}

				data "observe_board" "first" {
					id = observe_board.first.id
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_board.first", "id"),
					resource.TestCheckResourceAttrSet("data.observe_board.first", "oid"),
					resource.TestCheckResourceAttr("data.observe_board.first", "name", randomPrefix),
				),
			},
		},
	})
}
