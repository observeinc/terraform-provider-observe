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
	linkConfigPreamble = configPreamble + datastreamConfigPreamble + `
		resource "observe_dataset" "a" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-A"

			inputs = { "test" = observe_datastream.test.dataset }

			stage {
				pipeline = <<-EOF
					filter false
					colmake key:"test"
				EOF
			}
		}

		resource "observe_dataset" "b" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-B"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					makeresource primarykey(key)
				EOF
			}
		}`
)

func TestAccObserveLinkCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_link.example", "workspace"),
					resource.TestCheckResourceAttr("observe_link.example", "fields.0", "key"),
					resource.TestCheckResourceAttr("observe_link.example", "label", randomPrefix),
				),
			},
			{
				// if source and target column name in a field is the same, we can elide target
				// this should result in no diff
				PlanOnly: true,
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
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

func TestAccObserveLinkErrors(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_datastream.test.dataset
					target    = observe_datastream.test.dataset
					fields    = ["test"]
					label     = "%[1]s-link"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(".*not present.*"),
			},
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_datastream.test.dataset
					target    = observe_datastream.test.dataset
					fields    = ["OBSERVATION_KIND:FIELDS"]
					label     = "%[1]s-link"
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
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-link"
				}

				data "observe_link" "verify_example" {
					source = observe_link.example.source
					target = observe_link.example.target
					fields = observe_link.example.fields
				}
				`, randomPrefix),
			},
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-link"
				}

				data "observe_link" "check_example" {
					source = observe_link.example.source
					target = observe_link.example.target
					fields = observe_link.example.fields
				}

				resource "observe_dataset" "c" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-C"

					inputs = { "a" = observe_link.example.source }

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}

				data "observe_link" "check_propagated" {
					source = observe_dataset.c.oid
					target = observe_dataset.b.oid
					fields = observe_link.example.fields
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
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "first" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-first"
				}

				resource "observe_link" "second" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key"]
					label     = "%[1]s-second"
				}

				data "observe_link" "verify_first" {
					source = observe_link.first.source
					target = observe_link.first.target
					fields = observe_link.first.fields
				}

				data "observe_link" "verify_second" {
					source = observe_link.second.source
					target = observe_link.second.target
					fields = observe_link.second.fields
				}
				`, randomPrefix),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestLinkSuppression(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	/* Without suppression, reapplying this config repeatedly will never converge.
	 *
	 * This config creates datasets: A, B and C, where B and C are derived from A.
	 * We then create two links: A -> B and A -> C
	 *
	 * On a first apply, the links case a version increment in A, which causes
	 * the links to be reapplied.
	 *
	 * If we reapply links whenever a source or target change version, this
	 * cycle will repeat itself, and our infrastructure will never converge to
	 * the intended state.
	 *
	 * We currently avoid this by using `lastSaved` timestamp instead of `version`
	 */
	config := fmt.Sprintf(linkConfigPreamble+`
		resource "observe_dataset" "c" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-C"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					makeresource primarykey(key)
				EOF
			}
		}

		resource "observe_link" "a_to_b" {
			workspace = data.observe_workspace.default.oid
			source    = observe_dataset.a.oid
			target    = observe_dataset.b.oid
			fields    = ["key"]
			label     = "%[1]s-b"

		}

		resource "observe_link" "a_to_c" {
			workspace = data.observe_workspace.default.oid
			source    = observe_dataset.a.oid
			target    = observe_dataset.c.oid
			fields    = ["key"]
			label     = "%[1]s-c"
		}`, randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// expect to converge on first pass.
				ExpectNonEmptyPlan: false,
				Config:             config,
			},
		},
	})
}

func TestAccOB5629(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a normal link
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_link.example", "workspace"),
					resource.TestCheckResourceAttr("observe_link.example", "fields.0", "key"),
					resource.TestCheckResourceAttr("observe_link.example", "label", randomPrefix),
				),
			},
			// Change the link so it returns an error status. Expect an error when a link is updated.
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:notexist"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`.*failed to update foreign key: The field "notexist" is not present in the dataset.*`),
			},
			// Reapply the link. Expect no diff and for the fields to have been updated.
			// Expect no error when this happens, otherwise a user can't update a erroring link.
			// Display a warning when a link with an error status is read (not checked below).
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:notexist"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_link.example", "fields.0", "key:notexist"),
				),
			},
			// Update the link with an error status. Expect no error, otherwise a user can't update a erroring link.
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
				resource "observe_link" "example" {
					workspace = data.observe_workspace.default.oid
					source    = observe_dataset.a.oid
					target    = observe_dataset.b.oid
					fields    = ["key:key"]
					label     = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_link.example", "fields.0", "key"),
				),
			},
		},
	})
}
