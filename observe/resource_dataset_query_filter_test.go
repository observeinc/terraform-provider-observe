package observe

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccObserveDatasetQueryFilter(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test.oid
					label       = "%[1]s-filter"
					description = "Test filter"
					filter      = "1 = 2 or 'something' = 'something else'"
					disabled    = false
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "label", randomPrefix+"-filter"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "description", "Test filter"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "filter", "1 = 2 or 'something' = 'something else'"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "disabled", "false"),
					resource.TestCheckResourceAttrSet("observe_dataset_query_filter.test", "oid"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "errors.#", "0"),
				),
			},
			// Test update
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test.oid
					label       = "%[1]s-filter-updated"
					description = "Test filter for sensitive data"
					filter      = "OBSERVATION_KIND = 'sensitive'"
					disabled    = true
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "label", randomPrefix+"-filter-updated"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "description", "Test filter for sensitive data"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "filter", "OBSERVATION_KIND = 'sensitive'"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "errors.#", "0"),
				),
			},
			// Test 2 filters
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test.oid
					label       = "%[1]s-filter"
					description = "Test filter for sensitive data"
					filter      = "OBSERVATION_KIND = 'sensitive'"
					disabled    = false
				}
				resource "observe_dataset_query_filter" "test2" {
					dataset     = observe_dataset.test.oid
					label       = "%[1]s-filter2"
					description = "Filter out everything"
					filter      = "false"
					disabled    = false
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test2", "label", randomPrefix+"-filter2"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test2", "filter", "false"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test2", "disabled", "false"),
					resource.TestCheckResourceAttrSet("observe_dataset_query_filter.test2", "oid"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test2", "errors.#", "0"),
				),
			},
			// Test dataset breaking change
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							drop_col OBSERVATION_KIND
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test.oid
					label       = "%[1]s-filter"
					description = "Test filter for sensitive data"
					filter      = "OBSERVATION_KIND = 'sensitive'"
					disabled    = false
				}
				`, randomPrefix),
				// need to refresh state to see the query filter error (since there was no diff for it in this step),
				// so checks are in the following step
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("observe_dataset_query_filter.test", "errors.0", func(val string) error {
						if !strings.Contains(val, "the field \"OBSERVATION_KIND\" does not exist") {
							return fmt.Errorf("expected error containing 'the field \"OBSERVATION_KIND\" does not exist', got %s", val)
						}
						return nil
					}),
				),
			},
		},
	})
}

// Ensure default values are set correctly for optional fields
func TestAccObserveDatasetQueryFilterMinimal(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset_query_filter" "minimal" {
					dataset = observe_datastream.test.dataset
					label   = "%[1]s"
					filter  = "true"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset_query_filter.minimal", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.minimal", "filter", "true"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.minimal", "disabled", "false"),
					resource.TestCheckResourceAttr("observe_dataset_query_filter.minimal", "description", ""),
				),
			},
		},
	})
}

// Test that changing the dataset ID results in a new query filter resource
func TestAccObserveDatasetQueryFilterDatasetIDChange(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	queryFilterId := ""

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test1" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset1"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset" "test2" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset2"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test1.oid
					label       = "%[1]s-filter"
					filter      = "true"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dataset_query_filter.test", "label", randomPrefix+"-filter"),
					func(s *terraform.State) error {
						queryFilterId = s.RootModule().Resources["observe_dataset_query_filter.test"].Primary.ID
						return nil
					},
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test1" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset1"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset" "test2" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-dataset2"
					inputs = { "test" = observe_datastream.test.dataset }
					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}
				}
				resource "observe_dataset_query_filter" "test" {
					dataset     = observe_dataset.test2.oid
					label       = "%[1]s-filter"
					filter      = "true"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						if s.RootModule().Resources["observe_dataset_query_filter.test"].Primary.ID == queryFilterId {
							return fmt.Errorf("dataset query filter ID did not change")
						}
						return nil
					},
				),
			},
		},
	})
}
