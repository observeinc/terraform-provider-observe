package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveReferenceTable(t *testing.T) {
	randomPrefix1 := acctest.RandomWithPrefix("tf")
	randomPrefix2 := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source = "testdata/reference_table.csv"
					checksum = filemd5("testdata/reference_table.csv")
					description = "test"
					primary_key = ["state_code"]
					label_field = "state"
				}
				`, randomPrefix1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix1),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", "test"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.0", "state_code"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "state"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "checksum", "93dc3e9f2c6e30cd956eb062c18112eb"),
				),
			},
			// Changing the file will use PUT
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					description = "hello world!"
					primary_key = ["col1", "col2"]
					label_field = "col3"
				}
				`, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix2),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", "hello world!"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.0", "col1"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.1", "col2"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "col3"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "checksum", "891217caed9a1c2b325f23f418afbde5"),
				),
			},
			// Changing just metadata will use PATCH
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					description = "updated description"
					primary_key = ["col2"]
					label_field = "col3"
				}
				`, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix2),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", "updated description"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.0", "col2"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "col3"),
				),
			},
			// Ensure removing fields works using PATCH
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					label_field = "col3"
				}
				`, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix2),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", ""),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.#", "0"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "col3"),
				),
			},
		},
	})
}
