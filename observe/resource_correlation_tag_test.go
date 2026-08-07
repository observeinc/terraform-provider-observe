package observe

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
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

// TestCorrelationTagAdoptExisting verifies that creating a correlation tag that
// already exists on the dataset -- e.g. because it was created in the UI, or by
// another `apply` -- adopts the existing tag instead of failing. A correlation tag
// has no ID of its own; it's identified entirely by its (dataset, tag, path), so an
// existing match is already in the desired state.
func TestCorrelationTagAdoptExisting(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	tag := fmt.Sprintf("%s-key.name", randomPrefix)

	var datasetID string

	correlationTagConfig := fmt.Sprintf(linkConfigPreamble+`
				resource "observe_correlation_tag" "example" {
				name = "%[1]s-key.name"
				dataset = observe_dataset.a.oid
				column = "key"
			}`, randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Get the dataset created (and the provider configured) without
				// declaring the correlation tag resource yet, so the next step's
				// PreConfig can create the tag out-of-band -- simulating a tag
				// that was already created outside of this Terraform config.
				Config: fmt.Sprintf(linkConfigPreamble, randomPrefix),
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["observe_dataset.a"]
					if !ok {
						return fmt.Errorf("observe_dataset.a not found in state")
					}
					datasetID = rs.Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					client := testAccProvider.Meta().(*observe.Client)
					if err := client.CreateCorrelationTag(context.Background(), datasetID, tag, gql.LinkFieldInput{
						Column: "key",
						Path:   stringPtr(""),
					}); err != nil {
						t.Fatalf("failed to pre-create correlation tag out-of-band: %s", err)
					}
				},
				Config: correlationTagConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", tag),
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "column", "key"),
					resource.TestCheckResourceAttrSet("observe_correlation_tag.example", "dataset"),
				),
			},
			// Adopting the existing tag should leave no further diff.
			testAccPlanOnlyNoDriftStep(correlationTagConfig),
		},
	})
}
