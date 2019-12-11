package observe

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/mitchellh/mapstructure"
)

func TestAccObserveDatasetBasic(t *testing.T) {
	name := "observe_dataset.tf-acc-basic-dataset"
	resourceName := strings.Split(name, ".")[1]
	workspaceID, datasetID := testAccGetWorkspaceAndDataset(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDatasetConfig(resourceName, workspaceID, datasetID, "filter true"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "workspace", workspaceID),
					resource.TestCheckResourceAttr(name, "label", "dataset"),
					resource.TestCheckResourceAttr(name, "input.0.name", "0"),
					resource.TestCheckResourceAttr(name, "pipeline", "filter true"),
				),
			},
		},
	})
}

func testAccGetWorkspaceAndDataset(t *testing.T) (string, string) {
	client := sharedClient(t)
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

func testDatasetConfig(resourceID string, workspaceID string, inputID string, pipeline string) string {
	return fmt.Sprintf(`
	resource "observe_dataset" "%[1]s" {
		workspace = "%[2]s"
		input {
			dataset = "%[3]s"
		}
		pipeline = <<-EOF
			%[4]s
		EOF
	}`, resourceID, workspaceID, inputID, pipeline)
}
