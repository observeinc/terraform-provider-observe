package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

/*
func init() {
	resource.AddTestSweepers("observe_transform", &resource.Sweeper{
		Name: "observe_transform",
		F:    testSweepObserveDataset,
	})
}

func testSweepObserveDataset(r string) error {
	client, err := sharedClient()
	if err != nil {
		log.Printf("[ERROR] Failed to create Observe client: %s", err)
	}

	return nil
}
*/

func TestAccObserveDatasetBasic(t *testing.T) {
	workspaceID, _ := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDatasetConfig(workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "Test Dataset"),
				),
			},
		},
	})
}

func testAccGetWorkspaceAndDatasetID(t *testing.T) (string, string) {
	client, err := sharedClient()
	if err != nil {
		t.Fatal("failed to load client:", err)
	}

	datasets, err := client.ListDatasets()
	if err != nil {
		t.Fatal("failed to list datasets:", err)
	}

	if len(datasets) == 0 {
		t.Fatal("no datasets available")
	}

	return datasets[0].WorkspaceID, datasets[0].ID
}

func testDatasetConfig(workspaceID string) string {
	return fmt.Sprintf(`
	resource "observe_dataset" "first" {
		workspace = "%[1]s"
		name 	  = "Test Dataset"
		freshness = "1m"
		icon_url  = "input"
	}`, workspaceID)
}
