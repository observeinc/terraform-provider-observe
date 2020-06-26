package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDataset(t *testing.T) {
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
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "Observation"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dataset.observation", "id"),
				),
			},
		},
	})
}

func TestAccObserveSourceDatasetNotFound(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_workspace" "kubernetes" {
				  name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "%s"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(randomPrefix),
			},
		},
	})
}
