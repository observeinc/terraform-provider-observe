package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveForeignKeyCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	t.Skip("skipping due to OBS-2267.")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = data.observe_dataset.observation.oid
					target    = data.observe_dataset.observation.oid
					fields    = ["OBSERVATION_KIND:OBSERVATION_KIND"]
					label     = "%s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_fk.example", "workspace"),
					resource.TestCheckResourceAttr("observe_fk.example", "fields.0", "OBSERVATION_KIND"),
					resource.TestCheckResourceAttr("observe_fk.example", "label", randomPrefix),
				),
			},
			{
				// if source and target column name in a field is the same, we can elide target
				// this should result in no diff
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = data.observe_dataset.observation.oid
					target    = data.observe_dataset.observation.oid
					fields    = ["OBSERVATION_KIND"]
					label     = "%s"
				}
				`, randomPrefix),
			},
		},
	})
}

func TestAccObserveForeignKeyErrors(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = data.observe_dataset.observation.oid
					target    = data.observe_dataset.observation.oid
					fields    = ["test"]
					label     = "%[1]s-fk"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(".*not present in the dataset"),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = data.observe_dataset.observation.oid
					target    = data.observe_dataset.observation.oid
					fields    = ["OBSERVATION_KIND:FIELDS"]
					label     = "%[1]s-fk"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(".*cannot be used as a key"),
			},
		},
	})
}
