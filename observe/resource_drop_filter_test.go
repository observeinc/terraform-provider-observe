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
	ingestFilterConfigPreabmle = configPreamble + datastreamConfigPreamble
)

func TestIngestFilterDropRateTooLow(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(ingestFilterConfigPreabmle+`
				resource "observe_drop_filter" "example" {
					workspace = data.observe_workspace.default.oid
					name = "%[1]s"
					pipeline = "filter FIELDS.x ~ y"
					source_dataset= observe_datastream.test.dataset
					drop_rate = -0.1
					enabled = true
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile("dropRate must be between 0.0 and 1.0"),
			},
		},
	})
}

func TestIngestFilterInvalidFunction(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(ingestFilterConfigPreabmle+`
				resource "observe_drop_filter" "example" {
					workspace = data.observe_workspace.default.oid
					name = "%[1]s-filter"
					pipeline = "filter FIELDS.x > parse_duration(\"2h 30m\")"
					source_dataset= observe_datastream.test.dataset
					drop_rate = 0.99
					enabled = true
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(".*the filter contains one or more functions not supported.*"),
			},
		},
	})
}
func TestIngestFilterCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(ingestFilterConfigPreabmle+`
				resource "observe_drop_filter" "example" {
					workspace = data.observe_workspace.default.oid
					name = "%[1]s-filter"
					pipeline = "filter FIELDS.x ~ y"
					source_dataset= observe_datastream.test.dataset
					drop_rate = 0.99
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_drop_filter.example", "name", randomPrefix+"-filter"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "drop_rate", "0.99"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "pipeline", "filter FIELDS.x ~ y"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "enabled", "true"),
					resource.TestCheckResourceAttrPair("observe_drop_filter.example", "source_dataset", "observe_datastream.test", "dataset"),
				),
			},
			{
				Config: fmt.Sprintf(ingestFilterConfigPreabmle+`
				resource "observe_drop_filter" "example" {
					workspace = data.observe_workspace.default.oid
					name = "%[1]s-filter-1"
					pipeline = "filter FIELDS.x ~ x"
					source_dataset= observe_datastream.test.dataset
					drop_rate = 0.88
					enabled = false
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_drop_filter.example", "name", randomPrefix+"-filter-1"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "drop_rate", "0.88"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "pipeline", "filter FIELDS.x ~ x"),
					resource.TestCheckResourceAttr("observe_drop_filter.example", "enabled", "false"),
					resource.TestCheckResourceAttrPair("observe_drop_filter.example", "source_dataset", "observe_datastream.test", "dataset"),
				),
			},
		},
	})
}
