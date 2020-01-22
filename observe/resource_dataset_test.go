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
				Config: fmt.Sprintf(`
				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_test_dataset"
					freshness = "1m"
					icon_url  = "input"
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "tf_test_dataset"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "1m0s"),
				),
			},
		},
	})
}

func TestAccObserveDatasetSchema(t *testing.T) {
	workspaceID, _ := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_test_schema"

					field { name = "column" }
					field {
						name = "number"
						type = "int64"
					}
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "tf_test_schema"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckResourceAttr("observe_dataset.first", "field.0.name", "column"),
				),
			},
		},
	})
}

func TestAccObserveDatasetUpdate(t *testing.T) {
	workspaceID, _ := testAccGetWorkspaceAndDatasetID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_test_update"
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "tf_test_update"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "tf_test_update"
					freshness = "1m"
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", "tf_test_update"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "1m0s"),
				),
			},
		},
	})
}

func TestAccObserveDatasetEmbeddedTransform(t *testing.T) {
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

					stage {
						input 	 = data.observe_dataset.observation.id
						pipeline = "filter true"
				  	}
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", datasetID),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "observe_dataset" "observation" {
					workspace = "%[1]s"
					name      = "Observation"
				}

				resource "observe_dataset" "first" {
					workspace = "%[1]s"
					name 	  = "some test dataset"
				}`, workspaceID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "workspace", workspaceID),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "stage"),
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

	for _, d := range datasets {
		if d.Config.Name == "Observation" {
			return d.WorkspaceID, d.ID
		}
	}
	t.Fatal("failed to find observation table")
	return "", ""
}
