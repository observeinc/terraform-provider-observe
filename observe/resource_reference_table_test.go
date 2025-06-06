package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveReferenceTable(t *testing.T) {
	t.Skip()
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
					source_file = "testdata/reference_table.csv"
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
					resource.TestCheckResourceAttrSet("observe_reference_table.example", "dataset"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.#", "5"),
					// implicit all string schema when no schema is provided
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.0.name", "rank"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.0.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.1.name", "state"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.1.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.2.name", "state_code"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.2.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.3.name", "2020_census"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.3.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.4.name", "percent_of_total"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.4.type", "string"),
				),
			},
			// Changing the file will use PUT
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source_file = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					description = "hello world!"
					primary_key = ["col1", "col2"]
					label_field = "col3"
				}
				`, randomPrefix1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix1),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", "hello world!"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.0", "col1"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.1", "col2"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "col3"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "checksum", "891217caed9a1c2b325f23f418afbde5"),
				),
			},
			// Changing just metadata will use PATCH
			// TODO: currently just label and description, API will support PATCHing primary_key and label_field soon
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source_file = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					description = "updated description"
					primary_key = ["col1", "col2"]
					label_field = "col3"
				}
				`, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "label", randomPrefix2),
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", "updated description"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.0", "col1"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "primary_key.1", "col2"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "label_field", "col3"),
				),
			},
			// Ensure removing fields works using PATCH
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source_file = "testdata/reference_table2.csv"
					checksum = filemd5("testdata/reference_table2.csv")
					primary_key = ["col1", "col2"]
					label_field = "col3"
				}
				`, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "description", ""),
				),
			},
		},
	})
}

func TestAccObserveReferenceTableSchema(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source_file = "testdata/reference_table.csv"
					checksum = filemd5("testdata/reference_table.csv")

					schema {
						name = "rank"
						type = "int64"
					}
					schema {
						name = "state"
						type = "string"
					}
					schema {
						name = "state_code"
						type = "string"
					}
					schema {
						name = "2020_census"
						type = "int64"
					}
					schema {
						name = "percent_of_total"
						type = "float64"
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.#", "5"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.0.name", "rank"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.0.type", "int64"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.1.name", "state"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.1.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.2.name", "state_code"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.2.type", "string"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.3.name", "2020_census"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.3.type", "int64"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.4.name", "percent_of_total"),
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.4.type", "float64"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "example" {
					label      = "%s"
					source_file = "testdata/reference_table.csv"
					checksum = filemd5("testdata/reference_table.csv")

					schema {
						name = "rank"
						type = "string"
					}
					schema {
						name = "state"
						type = "string"
					}
					schema {
						name = "state_code"
						type = "string"
					}
					schema {
						name = "2020_census"
						type = "int64"
					}
					schema {
						name = "percent_of_total"
						type = "float64"
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_reference_table.example", "schema.0.type", "string"),
				),
			},
		},
	})
}
