package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestCorrelationTagCreation(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", fmt.Sprintf("%s-key.name", randomPrefix)),
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "column", "key"),
					resource.TestCheckResourceAttrSet("observe_correlation_tag.example", "dataset"),
				),
			},
			// Using the same config, there should not be any diff.
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Making any change to the config should delete and recreate the tag (in-place update is not supported)
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name-2"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				Check: resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", fmt.Sprintf("%s-key.name-2", randomPrefix)),
			},
		},
	})
}
