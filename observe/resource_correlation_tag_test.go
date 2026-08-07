package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func TestCorrelationTagCreation(t *testing.T) {
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
			{
				ResourceName: "observe_correlation_tag.example",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["observe_correlation_tag.example"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					// Reconstruct the JSON ID from resource attributes
					datasetOid, _ := oid.NewOID(rs.Primary.Attributes["dataset"])
					params := correlationTagParameters{
						Dataset: datasetOid.Id,
						Tag:     rs.Primary.Attributes["name"],
						Path: gql.LinkFieldInput{
							Column: rs.Primary.Attributes["column"],
						},
					}
					if path, ok := rs.Primary.Attributes["path"]; ok && path != "" {
						params.Path.Path = &path
					}
					return constructCorrelationTagId(params.Dataset, params.Tag, params.Path), nil
				},
				ImportStateVerify: true,
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
