package observe

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// common to all configs
	defaultWorkspaceName = getenv("OBSERVE_WORKSPACE", "Default")
	configPreamble       = fmt.Sprintf(`
				data "observe_workspace" "default" {
					name = "%s"
				}`, defaultWorkspaceName)
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "72h0m0s"),
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
				Check: resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "72h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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

	resource.ParallelTest(t, resource.TestCase{
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "768h0m0s"),
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

// Test entity_tags field for datasets
func TestAccObserveDatasetEntityTags(t *testing.T) {
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

					entity_tags = {
						environment = "production"
						team        = "backend,frontend"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-dataset"),
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.environment", "production"),
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.team", "backend,frontend"),
				),
			},
			{
				// Update entity_tags
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

					entity_tags = {
						environment = "staging,production"
						region      = "us-west-2"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.environment", "production,staging"), // Backend sorts alphabetically
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.region", "us-west-2"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "entity_tags.team"),
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

					entity_tags = {
						note = "\"Team A, Inc\""
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.note", "\"Team A, Inc\""),
				),
			},
			{
				// Remove all entity_tags
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
					resource.TestCheckResourceAttr("observe_dataset.first", "entity_tags.%", "0"),
				),
			},
		},
	})
}
