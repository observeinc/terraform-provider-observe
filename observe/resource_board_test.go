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
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_board" "first" {
					dataset  = observe_datastream.test.dataset
					name     = "%[1]s"
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
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_board" "first" {
					dataset  = observe_datastream.test.dataset
					name     = "%[1]s-2"
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

// Test JSON attribute handles unresolved values.
func TestAccObserveBoardJSON(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
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

				resource "observe_board" "second" {
					dataset  = observe_datastream.test.dataset
					name     = "%[1]s"
					type     = "set"
					# on plan, value will be unresolved
					json     = jsonencode({
						"bla" = observe_board.first.id
					})
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_board.first", "dataset"),
					resource.TestCheckResourceAttr("observe_board.first", "name", randomPrefix),
				),
			},
		},
	})
}
