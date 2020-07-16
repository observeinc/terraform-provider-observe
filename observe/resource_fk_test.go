package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// common to all configs
	fkConfigPreamble = configPreamble + `
		resource "observe_dataset" "a" {
			workspace = data.observe_workspace.kubernetes.oid
			name      = "%[1]s-A"

			inputs = { "observation" = data.observe_dataset.observation.oid }

			stage {
				pipeline = <<-EOF
					filter false
					colmake key:"test"
				EOF
			}
		}

		resource "observe_dataset" "b" {
			workspace = data.observe_workspace.kubernetes.oid
			name      = "%[1]s-B"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					makeresource primarykey(key)
				EOF
			}
		}`
)

func TestAccObserveForeignKeyCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_fk.example", "workspace"),
					resource.TestCheckResourceAttr("observe_fk.example", "fields.0", "key"),
					resource.TestCheckResourceAttr("observe_fk.example", "label", randomPrefix),
				),
				// We need to apply twice, since the first apply of observe_fk
				// will update the source and target dataset versions
				ExpectNonEmptyPlan: true,
			},
			{
				// Reapply to converge to newer dataset versions
				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}
				`, randomPrefix),
			},
			{
				// if source and target column name in a field is the same, we can elide target
				// this should result in no diff
				PlanOnly: true,
				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s"
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
				Config: fmt.Sprintf(fkConfigPreamble+`
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
				Config: fmt.Sprintf(fkConfigPreamble+`
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
