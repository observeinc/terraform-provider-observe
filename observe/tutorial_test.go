package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccObserveDatasetBasic will capture the examples in our tutorial
func TestAccObserveDatasetBasic(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// simple case: one input, one stage.
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  	"observation" = data.observe_dataset.observation.oid
					}

					stage {
						input    = "observation"
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),

					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "icon_url"),

					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", "observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", "filter true\n"),
				),
			},
			{
				// you can elide the stage input if only one input is available
				// you can update dataset properties
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					icon_url  = "test-tube"
					freshness = "2m"

					inputs = {
					  	"observation" = data.observe_dataset.observation.oid
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
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
		},
	})
}
