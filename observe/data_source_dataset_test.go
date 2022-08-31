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
				Config: configPreamble,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dataset.observation", "id"),
				),
			},
		},
	})
}

func TestAccObserveSourceDatasetStage(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_dataset" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = { "observation" = data.observe_dataset.observation.oid }

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}
					}

					data "observe_dataset" "lookup_by_name" {
						workspace  = data.observe_workspace.default.oid
						name       = observe_dataset.a.name
						depends_on = [observe_dataset.a]
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_name", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_name", "stage.0.pipeline", "filter false\n"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
						resource "observe_dataset" "a" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s"

							inputs = { "observation" = data.observe_dataset.observation.oid }

							stage {
								pipeline = <<-EOF
									filter false
								EOF
							}
						}

						data "observe_dataset" "lookup_by_id" {
							id         = observe_dataset.a.id
							depends_on = [observe_dataset.a]
						}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dataset.lookup_by_id", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "stage.0.pipeline", "filter false\n"),
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
				Config: fmt.Sprintf(configPreamble+`
				data "observe_dataset" "missing" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(randomPrefix),
			},
		},
	})
}
