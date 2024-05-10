package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestCorrelationTagCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: linkConfigPreamble + `
					resource "observe_correlation_tag" "example" {
					name = "key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", "key.name"),
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "column", "key"),
					resource.TestCheckResourceAttrSet("observe_correlation_tag.example", "dataset"),
				),
			},
			// Using the same config, there should not be any diff.
			{
				Config: linkConfigPreamble + `
					resource "observe_correlation_tag" "example" {
					name = "key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Making any change to the config should delete and recreate the tag (in-place update is not supported)
				Config: linkConfigPreamble + `
					resource "observe_correlation_tag" "example" {
					name = "key.name-2"
					dataset = observe_dataset.a.oid
					column = "key"
				}`,
				Check: resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", "key.name-2"),
			},
			{
				// Removing the config should delete the tag
				Config: linkConfigPreamble,
				Check:  resource.TestCheckNoResourceAttr("observe_correlation_tag.example", "name"),
			},
		},
	})
}
