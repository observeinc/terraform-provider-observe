package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceBoard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				resource "observe_board" "first" {
					dataset  = data.observe_dataset.observation.oid
					name     = "Test"
					type     = "set"
					json     = "{}"
				}

				data "observe_board" "first" {
				  id = observe_board.first.id
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_board.first", "id"),
					resource.TestCheckResourceAttrSet("data.observe_board.first", "oid"),
					resource.TestCheckResourceAttr("data.observe_board.first", "name", "Test"),
				),
			},
		},
	})
}
