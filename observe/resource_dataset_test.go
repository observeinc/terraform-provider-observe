package observe

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func TestAccObserveDatasetNameValidationTooLong(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s%s"  # exceeds MaxNameLength

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {}
				}`, randomPrefix, strings.Repeat("a", MaxNameLength)),
				ExpectError: regexp.MustCompile("expected length of name to be.*"),
			},
		},
	})
}

func TestAccObserveDatasetNameValidationInvalidCharacter(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s with colon :"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile("expected value of name to not contain.*"),
			},
		},
	})
}

// Verify we can change dataset properties: e.g. name and freshness
func TestAccObserveDatasetUpdate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					acceleration_disabled = true
					acceleration_disabled_source = "view"

					stage {}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "acceleration_disabled_source", "view"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "rematerialization_mode"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-rename"
					freshness                        = "1m"
					on_demand_materialization_length = "48h39s"
					path_cost                        = "1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					data_table_view_state = jsonencode({viewType = "Auto"})
					acceleration_disabled = true
					acceleration_disabled_source = "view"

					stage {
						pipeline = <<-EOF
							make_col x:1
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-rename"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "1m0s"),
					resource.TestCheckResourceAttr("observe_dataset.first", "path_cost", "1"),
					// On demand mat length has a daily resolution
					// So whatever the user sets here, we will round up the amount of days
					// In this case, 48h39s is rounded up to 72h
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", "make_col x:1\n"),
					resource.TestCheckResourceAttr("observe_dataset.first", "acceleration_disabled", "true"),
					resource.TestCheckResourceAttr("observe_dataset.first", "data_table_view_state", "{\"viewType\":\"Auto\"}"),
					resource.TestCheckResourceAttr("observe_dataset.first", "acceleration_disabled_source", "view"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-rename"
					freshness                        = "1m"
					on_demand_materialization_length = "48h0m39s"
					path_cost                        = 1

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					acceleration_disabled = true
					acceleration_disabled_source = "view"

					stage {
						pipeline = <<-EOF
							make_col x:1
						EOF
					}
				}`, randomPrefix),
			},
		},
	})
}

// Changing input name should not break implicit stage reference to input
func TestAccObserveDatasetChangeInputName(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
		},
	})
}

// Changing stage name from default should not break implicit stage reference to stage
func TestAccObserveDatasetChangeStageName(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.input", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						alias    = "first"
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						input    = "test"
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						pipeline = <<-EOF
							union @first
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", "first"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.input", "test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.input", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.input", ""),
				),
			},
		},
	})
}

// TestAccObserveDatasetRedundantAliasInputNoPerpetualDiff is a full acceptance-test regression
// guard for the perpetual-diff bug fixed by flattenAndSetQuery's per-stage config fallback: a
// stage whose "input" explicitly names the immediately preceding stage's alias (a valid but
// redundant form) had that value silently cleared from state on every read, because the
// config-preserving fallback only applied to stage 0. Every subsequent plan then showed the
// value being re-added, forever. This creates the real redundant-alias-input scenario against
// a live backend and asserts the plan stays empty on a second, unchanged apply -- proving the
// diff doesn't come back.
func TestAccObserveDatasetRedundantAliasInputNoPerpetualDiff(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	config := fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
		resource "observe_dataset" "chained" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-chained"

			inputs = { "test" = observe_datastream.test.dataset }

			stage {
				alias    = "base_stage"
				input    = "test"
				pipeline = <<-EOF
					filter true
				EOF
			}

			// input explicitly names stage 0's alias -- the redundant form that used to be
			// cleared from state on every read.
			stage {
				input    = "base_stage"
				pipeline = <<-EOF
					filter true
				EOF
			}
		}
	`, randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.chained", "stage.0.alias", "base_stage"),
					resource.TestCheckResourceAttr("observe_dataset.chained", "stage.1.input", "base_stage"),
				),
			},
			// Re-plan must be empty: before the fix, flattenAndSetQuery cleared stage 1's
			// input on every read, so this step would catch the perpetual "+ input" diff
			// that motivated this fix.
			testAccPlanOnlyNoDriftStep(config),
		},
	})
}

// Verify we can coldrop if no downstream affected
func TestAccObserveDatasetSchemaChange(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = { "test" = observe_datastream.test.dataset }

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-2"

					inputs = { "first" = observe_dataset.first.oid }

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
			},
			{
				// coldrop with no downstream breakage
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = { "test" = observe_datastream.test.dataset }

					stage {
						pipeline = <<-EOF
							coldrop FIELDS
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-2"

					inputs = { "first" = observe_dataset.first.oid }

					stage {
						pipeline = <<-EOF
							colmake test:object(EXTRA.tags)
						EOF
					}
				}`, randomPrefix),
			},
			{
				// downstream with breakage
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = { "test" = observe_datastream.test.dataset }

					stage {
						pipeline = <<-EOF
							coldrop EXTRA
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-2"

					inputs = { "first" = observe_dataset.first.oid }

					stage {
						pipeline = <<-EOF
							colmake test:object(EXTRA.tags)
						EOF
					}
				}`, randomPrefix),
				ExpectError: newMultilineErrorRegexp(`field "EXTRA" does not exist`),
			},
			{
				// we should always have a diff when applying after error.
				// in this case, we know second dataset has less recent version
				// than one of its dependencies, so we force recomputation.
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = { "test" = observe_datastream.test.dataset }

					stage {
						pipeline = <<-EOF
							coldrop EXTRA
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-2"

					inputs = { "first" = observe_dataset.first.oid }

					stage {
						pipeline = <<-EOF
							colmake test:object(EXTRA.tags)
						EOF
					}
				}`, randomPrefix),
				// Since we do server-side validation (dry-run saveDataset) during the plan stage now,
				// the plan for observe_dataset.second will fail.
				ExpectError: newMultilineErrorRegexp(`field "EXTRA" does not exist`),
			},
		},
	})
}

// Verify configuration errors
func TestAccObserveDatasetErrors(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
						"other" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(`stage-0: input missing`),
			},
		},
	})
}

// Test edit-forward works when change is compatible
func TestAccObserveDatasetEditForward(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_col x: 1
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "rematerialization_mode"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					rematerialization_mode = "must_skip_rematerialization"
					stage {
							pipeline = <<-EOF
							make_col x: 2
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "rematerialization_mode", "must_skip_rematerialization"),
				),
			},
		},
	})
}

// Test that the provider's default rematerialization_mode is respected
func TestAccObserveDatasetDefaultRematerializationMode(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	// see TestAccObserveSourceDashboard_ExportWithBindings for context
	providerPreamble := `
		terraform {} # trick the testing framework into not mangling our config
		provider "observe" {
			default_rematerialization_mode = "must_skip_rematerialization"
		}
	`

	// Serial: overriding provider config mutates the shared testAccProvider
	// instance, so this cannot run alongside other tests.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND)
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "rematerialization_mode"),
				),
			},
			{
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
							pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(`The following dataset\(s\) will be rematerialized`),
			},
			{ // Check the provider-level rematerialization option can be overridden
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					rematerialization_mode = "rematerialize"
					stage {
							pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "rematerialization_mode", "rematerialize"),
				),
			},
		},
	})
}

// Test that a change fails if rematerialization would occur under edit-forward
func TestAccObserveDatasetEditForwardDryRun(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND)
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "rematerialization_mode"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					rematerialization_mode = "must_skip_rematerialization"
					stage {
							pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(`The following dataset\(s\) will be rematerialized`),
			},
		},
	})
}

// Test that a change rematerializes when incompatible with edit-forward
func TestAccObserveDatasetEditForwardNoDryRun(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND)
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "rematerialization_mode"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					rematerialization_mode = "skip_rematerialization"
					stage {
							pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "rematerialization_mode", "skip_rematerialization"),
				),
			},
		},
	})
}

// Ensures that with rematerialization_mode = must_skip_rematerialization, if any datasets
// would be rematerialized due to the change, we fail during the plan stage.
func TestAccObserveDatasetRematerializedDatasetsDuringPlan(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test-rematerialize" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-test-rematerialize"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND)
						EOF
					}
				}`, randomPrefix),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test-rematerialize" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-test-rematerialize"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					rematerialization_mode = "must_skip_rematerialization"
					stage {
						pipeline = <<-EOF
							make_resource primary_key(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}
				}`, randomPrefix),
				PlanOnly:    true, // we only want to test the plan stage here
				ExpectError: regexp.MustCompile(`The following dataset\(s\) will be rematerialized`),
			},
		},
	})
}

func TestAccObserveDatasetDescription(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		// We use a data source to read the value of description back in.
		// This assures us that the value is correctly set and read from
		// backend, rather than just being set in local state.
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.default.oid
					name 	    = "%[1]s-1"
					description = "test description"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}

				data "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	    = observe_dataset.first.name
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", "test description"),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", "test description"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.default.oid
					name 	    = "%[1]s-1"
					description = "updated"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}

				data "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	    = observe_dataset.first.name
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", "updated"),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", "updated"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.default.oid
					name 	    = "%[1]s-1"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}

				data "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	    = observe_dataset.first.name
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", ""),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", ""),
				),
			},
		},
	})
}

func TestAccObserveDatasetMultiInput(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							pick_col BUNDLE_TIMESTAMP, tags:FIELDS
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s second"

					inputs = {
						"test" = observe_datastream.test.dataset
						"first" = observe_dataset.first.oid
					}

					stage {
						alias    = "from_first"
						input    = "first"
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						input    = "test"
						pipeline = <<-EOF
							pick_col BUNDLE_TIMESTAMP, tags:FIELDS
							union @from_first
						EOF
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.second", "inputs.first"),
				),
			},
		},
	})
}

func TestAccObserveDatasetQuotedInputReference(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							pick_col BUNDLE_TIMESTAMP, tags:FIELDS
						EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s second"

					inputs = {
						"test" = observe_datastream.test.dataset
						"first" = observe_dataset.first.oid
					}

					stage {
						alias    = "from_first-123"
						input    = "first"
						pipeline = <<-EOF
							filter true
						EOF
					}

					stage {
						input    = "test"
						pipeline = <<-EOF
							pick_col BUNDLE_TIMESTAMP, tags:FIELDS
							union @"from_first-123"
						EOF
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.second", "inputs.first"),
				),
			},
		},
	})
}

func TestAccObserveDatasetUseIcebergStorageIntegration(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	if os.Getenv("CI") != "true" {
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's Snowflake database.")
	}

	// ! do not edit !
	// This ID is pre-created in the ENG terraform integration test tenant (127814973959).
	// The acc test will only run successfully against that tenant, which is OK for now.
	// It will be removed once we add support for storage integration as a terraform resource.
	//
	// This storage integration was created with the following aws config, for future reference:
	//     "externalId": "0b8aadee-6ff7-4f94-bcb0-4e4b61656c99",
	//     "iamRoleArn": "arn:aws:iam::723346149663:role/jyc-iceberg-test",
	//     "s3BaseUrl": "s3://jyc-observeinc/iceberg/terraform-integration-test/"
	storageIntegrationID := "42184117"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				data "observe_oid" "si" {
					id   = "%[1]s"
					type = "storageintegration"
				}

				resource "observe_dataset" "iceberg" {
					workspace              = data.observe_workspace.default.oid
					name                   = "%[2]s-iceberg"
					storage_integration = data.observe_oid.si.oid

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							// do nothing
						EOF
					}
				}`, storageIntegrationID, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.iceberg", "name", randomPrefix+"-iceberg"),
					resource.TestCheckResourceAttrSet("observe_dataset.iceberg", "storage_integration"),
					resource.TestCheckResourceAttr("observe_dataset.iceberg", "storage_integration", "o:::storageintegration:"+storageIntegrationID),
					resource.TestCheckResourceAttrSet("observe_dataset.iceberg", "oid"),
					resource.TestCheckResourceAttrSet("observe_dataset.iceberg", "inputs.test"),
				),
			},
		},
	})
}

// Ensures that invalid opal is caught during the plan stage.
func TestAccObserveDatasetBadOpalDuringPlan(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// need to create the datastream beforehand, so that the inputs are known at plan time below
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble, randomPrefix),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "bad_opal" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-bad-opal"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter nonexistent_column = "foo"
						EOF
					}
				}`, randomPrefix),
				PlanOnly:    true, // we only want to test the plan stage here
				ExpectError: regexp.MustCompile("the field \"nonexistent_column\" does not exist among fields"),
			},
		},
	})
}

// Tests that we're able to update a currently broken dataset through terraform
func TestAccObserveDatasetTestUpdateBroken(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a parent dataset with column "key1" and a child that depends on that column.
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "parent" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-parent"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_col key1:string("foo")
						EOF
					}
				}

				resource "observe_dataset" "child" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-child"

					inputs = {
						"parent" = observe_dataset.parent.oid
					}

					stage {
						input    = "parent"
						pipeline = <<-EOF
							filter key1 = "foo"
						EOF
					}
				}`, randomPrefix),
			},
			// Update the parent to replace column "key1" with "key2". This should break the child.
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "parent" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-parent"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_col key2:string("bar")
						EOF
					}
				}

				resource "observe_dataset" "child" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-child"

					inputs = {
						"parent" = observe_dataset.parent.oid
					}

					stage {
						input    = "parent"
						pipeline = <<-EOF
							filter key1 = "foo"
						EOF
					}
				}`, randomPrefix),
				// Because the version in the dataset oid triggers a SaveDataset call on child, we
				// should see the parent get successfully updated, but the child should cause an error.
				ExpectError: regexp.MustCompile("the field \"key1\" does not exist among fields"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.parent", "stage.0.pipeline", "make_col key2:string(\"bar\")\n"),
				),
			},
			// Ensure it's possible to fix the child despite it currently being in a broken state.
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "parent" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-parent"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_col key2:string("bar")
						EOF
					}
				}

				resource "observe_dataset" "child" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%[1]s-child"

					inputs = {
						"parent" = observe_dataset.parent.oid
					}

					stage {
						input    = "parent"
						pipeline = <<-EOF
							filter key2 = "bar"
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.child", "stage.0.pipeline", "filter key2 = \"bar\"\n"),
				),
			},
		},
	})
}

// Test deprecated entity_tags still works for optional rename compatibility.
func TestAccObserveDatasetEntityTagsDeprecated(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
				resource "observe_dataset" "first" {
					name = "%[1]s-dataset"

					inputs = {
						"test" = observe_datastream.test_no_ws.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					entity_tags = {
						environment = "production"
					}
				}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.environment", "production"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "object_tags.environment"),
				),
			},
			testAccPlanOnlyNoDriftStep(config),
		},
	})
}

// Test object_tags field for datasets
func TestAccObserveDatasetObjectTags(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					object_tags = {
						environment = "production"
						team        = "backend,frontend"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-dataset"),
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.environment", "production"),
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.team", "backend,frontend"),
				),
			},
			{
				// Update object_tags
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					object_tags = {
						environment = "staging,production"
						region      = "us-west-2"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.environment", "production,staging"), // Backend sorts alphabetically
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.region", "us-west-2"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "object_tags.team"),
				),
			},
			{
				// Test CSV escaping (value with comma)
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					object_tags = {
						note = "\"Team A, Inc\""
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.note", "\"Team A, Inc\""),
				),
			},
			{
				// Remove all object_tags
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "object_tags.%", "0"),
				),
			},
		},
	})
}

func TestAccObserveDatasetNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(datastreamNoWorkspacePreamble+`
				resource "observe_dataset" "no_ws" {
					name = "%[1]s-no-ws"

					inputs = {
						"test" = observe_datastream.test_no_ws.dataset
					}

					stage {}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.no_ws", "workspace"),
					resource.TestCheckResourceAttr("observe_dataset.no_ws", "name", randomPrefix+"-no-ws"),
				),
			},
			testAccPlanOnlyNoDriftStep(fmt.Sprintf(datastreamNoWorkspacePreamble+`
				resource "observe_dataset" "no_ws" {
					name = "%[1]s-no-ws"

					inputs = {
						"test" = observe_datastream.test_no_ws.dataset
					}

					stage {}
				}`, randomPrefix)),
		},
	})
}

// oidVersionedRegex matches a dataset oid that carries a version suffix, e.g.
// o:::dataset:12345/2024-03-15T10:22:00Z (legacy behavior, flag off).
var oidVersionedRegex = regexp.MustCompile(`^o:::dataset:\d+/.+$`)

// oidVersionFreeRegex matches a dataset oid with no version suffix, e.g.
// o:::dataset:12345 (produced on first write when omit-dataset-oid-version is on).
var oidVersionFreeRegex = regexp.MustCompile(`^o:::dataset:\d+$`)

// TestAccObserveDatasetOmitOIDVersionNewResources verifies that datasets created
// from scratch with the omit-dataset-oid-version flag enabled get a version-free
// oid, and that editing an upstream dataset does not cascade a new version onto
// its downstream dependents.
func TestAccObserveDatasetOmitOIDVersionNewResources(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	// The test framework emits our config verbatim only when it contains a
	// "terraform {" block; otherwise it mangles it by injecting resources.
	providerOn := `
		terraform {}
		provider "observe" {
			flags = "omit-dataset-oid-version"
		}
	`

	// datasetChain builds an A -> B chain where B's inputs reference A's oid.
	// aPipeline lets us mutate A between steps to exercise the (non-)cascade.
	datasetChain := func(aPipeline string) string {
		return fmt.Sprintf(providerOn+configPreamble+datastreamConfigPreamble+`
		resource "observe_dataset" "a" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-A"

			inputs = { "test" = observe_datastream.test.dataset }

			stage {
				pipeline = <<-EOF
					%[2]s
				EOF
			}
		}

		resource "observe_dataset" "b" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-B"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					filter true
				EOF
			}
		}`, randomPrefix, aPipeline)
	}

	var bOID string

	// Serial: overriding provider config mutates the shared testAccProvider
	// instance, so this cannot run alongside other tests.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: datasetChain("make_col x:1"),
				Check: resource.ComposeTestCheckFunc(
					// oids are written without a version suffix on first create.
					resource.TestMatchResourceAttr("observe_dataset.a", "oid", oidVersionFreeRegex),
					resource.TestMatchResourceAttr("observe_dataset.b", "oid", oidVersionFreeRegex),
					// B's reference to A resolves to A's version-free oid.
					resource.TestMatchResourceAttr("observe_dataset.b", "inputs.a", oidVersionFreeRegex),
					resource.TestCheckResourceAttrWith("observe_dataset.b", "oid", func(v string) error {
						bOID = v
						return nil
					}),
				),
			},
			{
				// Edit A. Without the flag this would bump A's oid version and
				// cascade a new version onto B; with the flag B must be untouched.
				Config: datasetChain("make_col x:2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("observe_dataset.a", "oid", oidVersionFreeRegex),
					resource.TestCheckResourceAttrWith("observe_dataset.b", "oid", func(v string) error {
						if v != bOID {
							return fmt.Errorf("downstream dataset oid changed (cascade not suppressed): was %q, now %q", bOID, v)
						}
						return nil
					}),
				),
			},
			// Re-plan must be empty: a frozen oid produces no phantom diffs.
			testAccPlanOnlyNoDriftStep(datasetChain("make_col x:2")),
		},
	})
}

// TestAccObserveDatasetOmitOIDVersionMigration exercises the customer upgrade
// path: datasets are created with the flag OFF (versioned oids in state), then
// the flag is toggled ON. Toggling must be a no-op (no plan diff), and a
// subsequent upstream edit must not cascade onto downstream dependents.
func TestAccObserveDatasetOmitOIDVersionMigration(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	providerOff := `
		terraform {}
		provider "observe" {}
	`
	providerOn := `
		terraform {}
		provider "observe" {
			flags = "omit-dataset-oid-version"
		}
	`

	datasetChain := func(provider, aPipeline string) string {
		return fmt.Sprintf(provider+configPreamble+datastreamConfigPreamble+`
		resource "observe_dataset" "a" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-A"

			inputs = { "test" = observe_datastream.test.dataset }

			stage {
				pipeline = <<-EOF
					%[2]s
				EOF
			}
		}

		resource "observe_dataset" "b" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-B"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					filter true
				EOF
			}
		}`, randomPrefix, aPipeline)
	}

	var aOID, bOID string

	// Serial: overriding provider config mutates the shared testAccProvider
	// instance, and this test also changes that config between steps.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Legacy behavior: created with the flag off, oids carry a version.
				Config: datasetChain(providerOff, "make_col x:1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("observe_dataset.a", "oid", oidVersionedRegex),
					resource.TestCheckResourceAttrWith("observe_dataset.a", "oid", func(v string) error {
						aOID = v
						return nil
					}),
					resource.TestCheckResourceAttrWith("observe_dataset.b", "oid", func(v string) error {
						bOID = v
						return nil
					}),
				),
			},
			{
				// Toggle the flag on with no config change. The existing versioned
				// oids are frozen in place, so this must be a pure no-op.
				Config: datasetChain(providerOn, "make_col x:1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("observe_dataset.a", "oid", func(v string) error {
						if v != aOID {
							return fmt.Errorf("upstream oid changed on flag toggle: was %q, now %q", aOID, v)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("observe_dataset.b", "oid", func(v string) error {
						if v != bOID {
							return fmt.Errorf("downstream oid changed on flag toggle: was %q, now %q", bOID, v)
						}
						return nil
					}),
				),
			},
			{
				// Edit A now that the flag is on. B must not get a cascaded version.
				Config: datasetChain(providerOn, "make_col x:2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("observe_dataset.b", "oid", func(v string) error {
						if v != bOID {
							return fmt.Errorf("downstream dataset oid changed (cascade not suppressed): was %q, now %q", bOID, v)
						}
						return nil
					}),
				),
			},
			testAccPlanOnlyNoDriftStep(datasetChain(providerOn, "make_col x:2")),
		},
	})
}

// TestFlattenAndSetQueryReferenceInputFiltering guards against the perpetual-diff bug where
// backend-derived inputs (auto-added from "^<link>" traversals, e.g. label(^"some link")) bleed
// into state and cause a never-converging planned removal on every subsequent plan. The filter
// does not key off InputRole: live testing against a real backend showed these auto-added inputs
// actually come back with InputRole=Data, not Reference, for ordinary dataset pipelines (Reference
// is only used by a couple of narrow, unrelated backend mechanisms). Instead it mirrors what this
// provider's own write path (newQuery/appendReferencedInputs) would ever send for a stage: the
// stage's default input (API convention: always index 0), or one explicitly "@name"-referenced in
// that stage's pipeline. Anything else can only have come from the backend itself.
func TestFlattenAndSetQueryReferenceInputFiltering(t *testing.T) {
	const (
		dataID = "11111"
		refID1 = "22222"
		refID2 = "33333"
	)
	stageID := "stage-0"
	refName1 := "Ref Dataset from usage of some link"
	refName2 := "Other Ref Dataset from usage of other link"

	// Simulates the backend response: one user-declared default input (index 0) and two
	// backend-derived inputs auto-added from the "^<link>" traversals in the pipeline. Note
	// InputRole is Data for all three, matching what a real backend actually returns for this
	// case -- the filter must not rely on it.
	buildStages := func() []gql.StageQuery {
		return []gql.StageQuery{{
			Id:       stringPtr(stageID),
			Pipeline: `lookup ^"some link" | lookup ^"other link"`,
			Input: []gql.StageQueryInputInputDefinition{
				{InputName: "data_input", InputRole: gql.InputRoleData, DatasetId: stringPtr(dataID)},
				{InputName: refName1, InputRole: gql.InputRoleData, DatasetId: stringPtr(refID1)},
				{InputName: refName2, InputRole: gql.InputRoleData, DatasetId: stringPtr(refID2)},
			},
		}}
	}

	t.Run("backend-derived inputs absent from config are not written to state", func(t *testing.T) {
		// Config declares only the default input; neither derived input is present.
		data := schema.TestResourceDataRaw(t, resourceDataset().Schema, map[string]interface{}{
			"name":   "test",
			"inputs": map[string]interface{}{"data_input": oid.OID{Type: oid.TypeDataset, Id: dataID}.String()},
			"stage":  []interface{}{map[string]interface{}{"pipeline": "filter true", "alias": "", "input": "", "output_stage": false}},
		})
		if _, err := flattenAndSetQuery(data, buildStages(), stageID, false); err != nil {
			t.Fatalf("flattenAndSetQuery: %v", err)
		}
		inputs := data.Get("inputs").(map[string]interface{})
		if len(inputs) != 1 {
			t.Errorf("expected exactly 1 input in state, got %d: %v", len(inputs), inputs)
		}
	})

	t.Run("only the derived input declared in config is kept; undeclared one is not added", func(t *testing.T) {
		// Config declares refName1 but not refName2; both come back from the backend.
		// This covers the backward-compat case where a user explicitly declared one derived
		// input as a workaround — it must be preserved while the undeclared one is still filtered.
		data := schema.TestResourceDataRaw(t, resourceDataset().Schema, map[string]interface{}{
			"name": "test",
			"inputs": map[string]interface{}{
				"data_input": oid.OID{Type: oid.TypeDataset, Id: dataID}.String(),
				refName1:     oid.OID{Type: oid.TypeDataset, Id: refID1}.String(),
			},
			"stage": []interface{}{map[string]interface{}{"pipeline": "filter true", "alias": "", "input": "", "output_stage": false}},
		})
		if _, err := flattenAndSetQuery(data, buildStages(), stageID, false); err != nil {
			t.Fatalf("flattenAndSetQuery: %v", err)
		}
		inputs := data.Get("inputs").(map[string]interface{})
		if _, ok := inputs[refName1]; !ok {
			t.Errorf("explicitly-declared derived input %q should be kept in state", refName1)
		}
		if _, ok := inputs[refName2]; ok {
			t.Errorf("undeclared derived input %q should not appear in state", refName2)
		}
		if len(inputs) != 2 {
			t.Errorf("expected exactly 2 inputs in state, got %d: %v", len(inputs), inputs)
		}
	})

	t.Run("a non-default input explicitly @-referenced in the pipeline is always kept, even undeclared", func(t *testing.T) {
		// "joined" is not the stage's default (it's not index 0) but the pipeline binds it by
		// name via "@joined" -- exactly what this provider's own write path would send. It must
		// never be filtered, regardless of whether config declares it, because it is not
		// backend-only: the provider can and does write it itself.
		joinedID := "44444"
		stages := []gql.StageQuery{{
			Id:       stringPtr(stageID),
			Pipeline: `join on(a = @joined.b), c:@joined.c`,
			Input: []gql.StageQueryInputInputDefinition{
				{InputName: "data_input", InputRole: gql.InputRoleData, DatasetId: stringPtr(dataID)},
				{InputName: "joined", InputRole: gql.InputRoleData, DatasetId: stringPtr(joinedID)},
			},
		}}
		data := schema.TestResourceDataRaw(t, resourceDataset().Schema, map[string]interface{}{
			"name":   "test",
			"inputs": map[string]interface{}{"data_input": oid.OID{Type: oid.TypeDataset, Id: dataID}.String()},
			"stage":  []interface{}{map[string]interface{}{"pipeline": "filter true", "alias": "", "input": "", "output_stage": false}},
		})
		if _, err := flattenAndSetQuery(data, stages, stageID, false); err != nil {
			t.Fatalf("flattenAndSetQuery: %v", err)
		}
		inputs := data.Get("inputs").(map[string]interface{})
		if _, ok := inputs["joined"]; !ok {
			t.Errorf("@-referenced input %q should be kept in state even though config doesn't declare it", "joined")
		}
	})
}

// TestAccObserveDatasetLinkDerivedInputNoPerpetualDiff is a full acceptance-test regression
// guard for the perpetual-diff bug fixed by flattenAndSetQuery's BackendOnly filtering: a
// dataset whose pipeline traverses a link via "^<link label>" (here label(^"...")) gets an
// implicit input named "<target> from usage of <link label>" auto-added by the backend on
// every save. Before the fix, that input round-tripped into Terraform state even though the
// config never declares it, so every subsequent plan showed it being removed, forever. This
// creates the real link-derived scenario against a live backend and asserts the plan stays
// empty on a second, unchanged apply -- proving the diff doesn't come back.
func TestAccObserveDatasetLinkDerivedInputNoPerpetualDiff(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	config := fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
		resource "observe_dataset" "target" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-target"

			inputs = { "test" = observe_datastream.test.dataset }

			stage {
				pipeline = <<-EOF
					colmake key:"k1", label_val:"target-label"
					set_label label_val
				EOF
			}
		}

		resource "observe_dataset" "source" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-source"

			inputs = {
				"test"   = observe_datastream.test.dataset
				"target" = observe_dataset.target.oid
			}

			stage {
				input = "test"
				pipeline = <<-EOF
					filter false
					colmake key:"k1"
					makeresource primarykey(key)
					set_link "%[1]s-link", key:@"target".key
				EOF
			}
		}

		resource "observe_link" "source_to_target" {
			workspace = data.observe_workspace.default.oid
			source    = observe_dataset.source.oid
			target    = observe_dataset.target.oid
			fields    = ["key:key"]
			label     = "%[1]s-link"
		}

		// Traverses the link via label(^"<link label>") without declaring the target as an
		// input -- this is exactly the pattern that used to leave a "<target> from usage of
		// <link label>" input stuck in state, undeclared and perpetually removed on plan.
		resource "observe_dataset" "consumer" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-consumer"

			inputs = { "source" = observe_dataset.source.oid }

			stage {
				input = "source"
				pipeline = <<-EOF
					make_col linked_label:label(^"%[1]s-link")
				EOF
			}

			depends_on = [observe_link.source_to_target]
		}
	`, randomPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.consumer", "oid"),
					// The backend-derived "<target> from usage of <link label>" input must
					// not appear in state: only the explicitly declared "source" input should.
					resource.TestCheckResourceAttr("observe_dataset.consumer", "inputs.%", "1"),
					resource.TestCheckResourceAttrSet("observe_dataset.consumer", "inputs.source"),
				),
			},
			// Re-plan must be empty: the backend re-derives the same link input on every
			// save, so if it ever leaked back into state this step would catch the perpetual
			// removal diff that motivated this fix.
			testAccPlanOnlyNoDriftStep(config),
		},
	})
}

