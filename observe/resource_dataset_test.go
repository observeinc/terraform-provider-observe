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

					stage {}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-1"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "path_cost"),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "on_demand_materialization_length"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "acceleration_disabled", "false"),
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

					acceleration_disabled = true
					data_table_view_state = jsonencode({viewType = "Auto"})
					rematerialization_mode = "skip_rematerialization"

					stage {
					  pipeline = <<-EOF
					  	filter true
					  EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix+"-rename"),
					resource.TestCheckResourceAttr("observe_dataset.first", "freshness", "1m0s"),
					resource.TestCheckResourceAttr("observe_dataset.first", "path_cost", "1"),
					resource.TestCheckResourceAttr("observe_dataset.first", "on_demand_materialization_length", "48h0m39s"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "acceleration_disabled", "true"),
					resource.TestCheckResourceAttr("observe_dataset.first", "data_table_view_state", "{\"viewType\":\"Auto\"}"),
					resource.TestCheckResourceAttr("observe_dataset.first", "rematerialization_mode", "skip_rematerialization"),
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

					stage {
					  pipeline = <<-EOF
					  	filter true

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
					  	colmake test:EXTRA.tags
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
					  	colmake test:EXTRA.tags
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
					  	colmake test:EXTRA.tags
					  EOF
					}
				}`, randomPrefix),
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
