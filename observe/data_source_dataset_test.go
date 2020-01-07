package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccObserveSourceDatasetBasic(t *testing.T) {
	workspaceID, datasetID := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testSourceDatasetConfig(workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dataset.observation", "id", datasetID),
				),
			},
		},
	})
}

func testSourceDatasetConfig(workspaceID string) string {
	return fmt.Sprintf(`
	data "observe_dataset" "observation" {
	  workspace = "%[1]s"
      name      = "Observation"
	}`, workspaceID)
}
