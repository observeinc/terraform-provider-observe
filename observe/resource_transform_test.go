package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccObserveTransformBasic(t *testing.T) {
	workspaceID, datasetID := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_dataset" "observation" {
					workspace = "%[1]s"
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "some test dataset"
				}

				resource "observe_transform" "first" {
					dataset = observe_dataset.first.id

					stage {
						input 	 = data.observe_dataset.observation.id
						pipeline = "filter true"
				  	}
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_transform.first", "stage.0.input", datasetID),
				),
			},
		},
	})
}

func TestAccObserveTransformEmptyReference(t *testing.T) {
	workspaceID, datasetID := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_dataset" "observation" {
					workspace = "%[1]s"
					name      = "Observation"
				}

				// no transform is associated to this dataset
				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_first"

					field { name = "OBSERVATION_KIND" }
				}

				resource "observe_dataset" "second" {
					workspace = "%[1]s"
					name 	  = "tf_reference_first"
				}

				// we should be able to reference the first dataset
				resource "observe_transform" "second" {
					dataset = observe_dataset.second.id

					references = {
						test = observe_dataset.first.id
					}

					stage {
						input 	 = data.observe_dataset.observation.id
						pipeline = <<-EOF
							filter false
							addfk "Test", OBSERVATION_KIND:@test.OBSERVATION_KIND
						EOF
				  	}
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_transform.second", "stage.0.input", datasetID),
				),
			},
		},
	})
}

// TestAccObserveTransformTeardown verifies we correctly delete transforms which reference each other
func TestAccObserveTransformTeardown(t *testing.T) {
	workspaceID, _ := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_dataset" "observation" {
					workspace = "%[1]s"
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_first"
				}

				resource "observe_dataset" "second" {
					workspace = "%[1]s"
					name 	  = "tf_second"

					field { name = "OBSERVATION_KIND" }
				}


				// first dataset references second
				resource "observe_transform" "first" {
					dataset = observe_dataset.first.id

					references = {
						test = observe_dataset.second.id
					}

					stage {
						input 	 = data.observe_dataset.observation.id
						pipeline = <<-EOF
							addfk "Test", OBSERVATION_KIND:@test.OBSERVATION_KIND
						EOF
				  	}
				}

				// second dataset uses first as input
				resource "observe_transform" "second" {
					dataset = observe_dataset.second.id

					stage {
						input 	 = observe_transform.first.id
						pipeline = <<-EOF
							filter false
						EOF
				  	}
				}`, workspaceID),
			},
			{
				Config: fmt.Sprintf(`
				data "observe_dataset" "observation" {
					workspace = "%[1]s"
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_first"
				}`, workspaceID),
			},
		},
	})
}
