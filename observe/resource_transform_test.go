package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccObserveTransformBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				data "observe_workspace" "kubernetes" {
					name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
					workspace = data.observe_workspace.kubernetes.id
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.id
					name 	  = "some test dataset"
				}

				resource "observe_transform" "first" {
					dataset = observe_dataset.first.id

					stage {
						input 	 = data.observe_dataset.observation.id
						pipeline = "filter true"
				  	}
				}`,
			},
		},
	})
}

func TestAccObserveTransformEmptyReference(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				data "observe_workspace" "kubernetes" {
					name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
					workspace = data.observe_workspace.kubernetes.id
					name      = "Observation"
				}

				// no transform is associated to this dataset
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.id
					name 	  = "tf_first"

					field { name = "OBSERVATION_KIND" }
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.id
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
				}`,
			},
		},
	})
}

// TestAccObserveTransformTeardown verifies we correctly delete transforms which reference each other
func TestAccObserveTransformTeardown(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				data "observe_workspace" "kubernetes" {
					name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
					workspace = data.observe_workspace.kubernetes.id
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.id
					name 	  = "tf_first"
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.id
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
				}`,
			},
			{
				Config: `
				data "observe_workspace" "kubernetes" {
					name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
					workspace = data.observe_workspace.kubernetes.id
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.id
					name 	  = "tf_first"
				}`,
			},
		},
	})
}
