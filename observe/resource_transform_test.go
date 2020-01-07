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
				Config: testTransformConfig(workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_transform.first", "stage.0.input", datasetID),
				),
			},
		},
	})
}

func testTransformConfig(workspaceID string) string {
	return fmt.Sprintf(`
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
	  	input = data.observe_dataset.observation.id
		pipeline = <<-EOF
		  filter true
		EOF
	  }
	}`, workspaceID)
}
