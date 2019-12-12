package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

/*
func init() {
	resource.AddTestSweepers("observe_dataset", &resource.Sweeper{
		Name: "observe_dataset",
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
	workspaceID, datasetID := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDatasetConfig(workspaceID, datasetID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.import", datasetID),
					resource.TestCheckResourceAttr("observe_dataset.second", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.second", "stage.0.import", datasetID),
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

func testDatasetConfig(workspaceID string, inputID string) string {
	return fmt.Sprintf(`
	resource "observe_dataset" "first" {
		workspace = "%[1]s"

		stage {
			import = "%[2]s"
			pipeline = <<-EOF
				filter true
			EOF
		}
	}

	resource "observe_dataset" "second" {
		workspace = "%[1]s"
		label     = "ny-label"

		stage {
			label  = "alt"
			import = "${observe_dataset.first.id}"
		}

		stage {
			pipeline = <<-EOF
				filter true
			EOF
		}

		stage {
			pipeline = <<-EOF
				filter false
			EOF
		}
	}`, workspaceID, inputID)
}
