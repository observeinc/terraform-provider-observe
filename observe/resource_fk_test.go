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

func TestAccOBS2432(t *testing.T) {
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
					fields    = ["key"]
					label     = "%[1]s-fk"
				}

				data "observe_fk" "verify_example" {
					source     = observe_dataset.a.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
					depends_on = [observe_fk.example]
				}
				`, randomPrefix),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "example" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-fk"
				}

				data "observe_fk" "check_example" {
					source     = observe_dataset.a.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
					depends_on = [observe_fk.example]
				}

				resource "observe_dataset" "c" {
					workspace = data.observe_workspace.kubernetes.oid
					name      = "%[1]s-C"

					inputs = { "a" = observe_dataset.a.oid }

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
					depends_on = [observe_fk.example]
				}

				data "observe_fk" "check_propagated" {
					source     = observe_dataset.c.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
				}
				`, randomPrefix),
			},
		},
	})
}

func TestAccOBS2110(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(fkConfigPreamble+`
				resource "observe_fk" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-first"
				}

				resource "observe_fk" "second" {
					workspace = data.observe_workspace.kubernetes.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-second"
				}

				data "observe_fk" "verify_first" {
					source     = observe_dataset.a.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
					depends_on = [observe_fk.first]
				}

				data "observe_fk" "verify_second" {
					source     = observe_dataset.a.oid
					target     = observe_dataset.b.oid
					fields     = ["key"]
					depends_on = [observe_fk.second]
				}
				`, randomPrefix),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
