package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccObserveDatasetBasic will capture the examples in our tutorial
func TestAccObserveDatasetBasic(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// simple case: one input, one stage.
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						input    = "test"
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),

					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),

					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "icon_url"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),

					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", "test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", "filter true\n"),
				),
			},
			{
				// you can elide the stage input if only one input is available
				// you can update dataset properties
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					icon_url  = "test-tube"
					freshness = "2m"
					path_cost = 50

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "path_cost", "50"),
					resource.TestCheckResourceAttr("observe_dataset.first", "icon_url", "test-tube"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "2m0s"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
			{
				// change it all back
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					icon_url  = "test-tube"
					freshness = "2m"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "icon_url", "test-tube"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "2m0s"),
					// ideally path_cost shouldn't be set, but more work than it is worth to make that happen
					resource.TestCheckResourceAttr("observe_dataset.first", "path_cost", "0"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
		},
	})
}
