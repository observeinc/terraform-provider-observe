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
	configPreamble = `
				data "observe_workspace" "kubernetes" {
					name = "Kubernetes"
				}

				data "observe_dataset" "observation" {
					workspace = data.observe_workspace.kubernetes.oid
					name      = "Observation"
				}`
)

// Verify we can change dataset properties: e.g. name and freshness
func TestAccObserveDatasetUpdate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
					resource.TestCheckNoResourceAttr("observe_dataset.first", "freshness"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.pipeline", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s-rename"
					freshness = "1m"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

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
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.alias", ""),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
			{
				PlanOnly: true,
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s-rename"
					freshness = "1m"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {
					  pipeline = <<-EOF
					  	filter true
					  EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.0.input", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "test" = data.observe_dataset.observation.oid
					}

					stage {
					  pipeline = <<-EOF
					  	filter true
					  EOF
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {
					  alias    = "first"
					  pipeline = <<-EOF
					  	filter true
					  EOF
					}

					stage {
					  input    = "observation"
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
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.1.input", "observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "stage.2.input", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-1"

					inputs = { "observation" = data.observe_dataset.observation.oid }

					stage {
					  pipeline = <<-EOF
					  	filter true
					  EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-1"

					inputs = { "observation" = data.observe_dataset.observation.oid }

					stage {
					  pipeline = <<-EOF
					  	coldrop FIELDS
					  EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.oid
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
				// downstream with breakage
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-1"

					inputs = { "observation" = data.observe_dataset.observation.oid }

					stage {
					  pipeline = <<-EOF
					  	coldrop EXTRA
					  EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-2"

					inputs = { "first" = observe_dataset.first.oid }

					stage {
					  pipeline = <<-EOF
					  	colmake test:EXTRA.tags
					  EOF
					}
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(`
graphql: errors in stage "stage-0": 1,14: \[\] non-existent path "EXTRA" among
fields \[BUNDLE_TIMESTAMP, OBSERVATION_KIND, FIELDS\]
`),
			},
			{
				// we should always have a diff when applying after error.
				// in this case, we know second dataset has less recent version
				// than one of its dependencies, so we force recomputation.
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-1"

					inputs = { "observation" = data.observe_dataset.observation.oid }

					stage {
					  pipeline = <<-EOF
					  	coldrop EXTRA
					  EOF
					}
				}

				resource "observe_dataset" "second" {
					workspace = data.observe_workspace.kubernetes.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%[1]s-1"

					inputs = { 
						"observation" = data.observe_dataset.observation.oid
						"other" = data.observe_dataset.observation.oid
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
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.kubernetes.oid
					name 	    = "%s"
					description = "test"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {
					  pipeline = <<-EOF
					    filter true
					  EOF
					}
				}

				data "observe_dataset" "first" {
					workspace  = data.observe_workspace.kubernetes.oid
					name 	   = "%[1]s"
					depends_on = [observe_dataset.first]
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", "test"),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.kubernetes.oid
					name 	    = "%s"
					description = "updated"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {
					  pipeline = <<-EOF
					    filter true
					  EOF
					}
				}

				data "observe_dataset" "first" {
					workspace  = data.observe_workspace.kubernetes.oid
					name 	   = "%[1]s"
					depends_on = [observe_dataset.first]
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", "updated"),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", "updated"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dataset" "first" {
					workspace   = data.observe_workspace.kubernetes.oid
					name 	    = "%s"

					inputs = {
					  "observation" = data.observe_dataset.observation.oid
					}

					stage {
					  pipeline = <<-EOF
					    filter true
					  EOF
					}
				}

				data "observe_dataset" "first" {
					workspace  = data.observe_workspace.kubernetes.oid
					name 	   = "%[1]s"
					depends_on = [observe_dataset.first]
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset.first", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset.first", "description", ""),
					resource.TestCheckResourceAttr("data.observe_dataset.first", "description", ""),
				),
			},
		},
	})
}
