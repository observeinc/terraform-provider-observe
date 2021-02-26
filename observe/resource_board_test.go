package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Verify we can change board
func TestAccObserveBoardUpdate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_board" "first" {
					dataset  = data.observe_dataset.observation.oid
					name     = "%s"
					type     = "set"
					json     = "{}"
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_board.first", "dataset"),
					resource.TestCheckResourceAttr("observe_board.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_board.first", "json", "{}"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_board" "first" {
					dataset  = data.observe_dataset.observation.oid
					name     = "%s-2"
					type     = "set"
					json     = jsonencode({
						sections = {
							card = {
								cardType = "section"
								title = "Summary"
								closed = false
							}
						}
					})
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_board.first", "dataset"),
					resource.TestCheckResourceAttr("observe_board.first", "name", fmt.Sprintf("%s-2", randomPrefix)),
					resource.TestCheckResourceAttrSet("observe_board.first", "json"),
				),
			},
		},
	})
}
