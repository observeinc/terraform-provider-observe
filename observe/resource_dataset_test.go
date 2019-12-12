package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/mitchellh/mapstructure"
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
	workspaceID, datasetID := testAccGetWorkspaceAndDataset(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDatasetConfig(workspaceID, datasetID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.second", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.second", "label", "ny-label"),
					resource.TestCheckResourceAttr("observe_dataset.first", "input.0.name", "i0"),
					resource.TestCheckResourceAttr("observe_dataset.first", "input.0.dataset", datasetID),
					resource.TestCheckResourceAttr("observe_dataset.second", "input.0.name", "i0"),
					resource.TestCheckResourceAttr("observe_dataset.second", "input.0.dataset", datasetID),
					resource.TestCheckResourceAttr("observe_dataset.second", "input.1.name", "alt"),
					resource.TestCheckResourceAttr("observe_dataset.second", "stage.0.pipeline", "filter true"),
					resource.TestCheckResourceAttr("observe_dataset.second", "stage.1.pipeline", "filter false"),
				),
			},
		},
	})
}

func testAccGetWorkspaceAndDataset(t *testing.T) (string, string) {
	client, err := sharedClient()
	if err != nil {
		t.Fatal("failed to load client:", err)
	}

	result, err := client.Run(`
	query {
		projects {
			id
			datasets {
				id
				label
			}
		}
	}`, nil)
	if err != nil {
		t.Fatal("request failed:", err)
	}

	workspaces := []struct {
		ID       string `json:"id"`
		Datasets []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"datasets"`
	}{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &workspaces,
	})

	if err := decoder.Decode(result["projects"]); err != nil {
		t.Fatal(err)
	}

	if len(workspaces) == 0 {
		t.Fatal("could not find a workspace to use for testing")
	} else if len(workspaces[0].Datasets) == 0 {
		t.Fatal("could not find a dataset to use as root for testing")
	}

	return workspaces[0].ID, workspaces[0].Datasets[0].ID
}

func testDatasetConfig(workspaceID string, inputID string) string {
	return fmt.Sprintf(`
	resource "observe_dataset" "first" {
		workspace = "%[1]s"

		input {
			dataset = "%[2]s"
		}

		stage {
			pipeline = <<-EOF
				filter false
			EOF
		}
	}

	resource "observe_dataset" "second" {
		workspace = "%[1]s"
		label     = "ny-label"

		input {
			dataset = "%[2]s"
		}

		input {
			name    = "alt"
			dataset = "${observe_dataset.first.id}"
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
