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

func TestAccObserveSourceDatasetResource(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")
	randomTablePrefix := strings.Replace(randomPrefix, "-", "_", -1)

	if os.Getenv("CI") != "true" {
		// The schemas "EXTERNAL" and "EXTERNAL2" were manually created for the provider CI Observe account.
		// While the test still passes even if the schema does not exist, it probably shouldn't, since Snowflake errors on the underlying commands.
		// The tables are entirely non-existent which is likewise not validated.
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's Snowflake database.")
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_source_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%s"

					schema = "EXTERNAL"
					table_name = "%s_TABLE_NAME"
					source_update_table_name = "%s_SOURCE_UPDATE_TABLE_NAME"
					valid_from_field = "TIMESTAMP"
					
					field {
						name = "BATCH_ID"
						type = "int64"
						sql_type = "NUMBER(38,0)"
						is_hidden = true
					}
					field {
						name = "TIMESTAMP"
						type = "timestamp"
						sql_type = "NUMBER(38,0)"
					}
					field {
						name = "LOG"
						type = "string"
						sql_type = "TEXT"
						is_enum = true
						is_searchable = false
						is_hidden = false
						is_const = false
						is_metric = false
					}
					field {
						name = "TAG"
						type = "variant"
						sql_type = "VARIANT"
					}
				}`, randomPrefix, randomTablePrefix, randomTablePrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_source_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "schema", "EXTERNAL"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "table_name", randomTablePrefix+"_TABLE_NAME"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "source_update_table_name", randomTablePrefix+"_SOURCE_UPDATE_TABLE_NAME"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "valid_from_field", "TIMESTAMP"),
					resource.TestCheckNoResourceAttr("observe_source_dataset.first", "freshness"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_source_dataset.first", "field.*", map[string]string{
						"name":          "BATCH_ID",
						"type":          "int64",
						"sql_type":      "NUMBER(38,0)",
						"is_hidden":     "true",
						"is_enum":       "false",
						"is_searchable": "false",
						"is_const":      "false",
						"is_metric":     "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("observe_source_dataset.first", "field.*", map[string]string{
						"name":          "LOG",
						"type":          "string",
						"sql_type":      "TEXT",
						"is_hidden":     "false",
						"is_enum":       "true",
						"is_searchable": "false",
						"is_const":      "false",
						"is_metric":     "false",
					}),
				),
			},
			{
				Config: configPreamble + fmt.Sprintf(`
				resource "observe_source_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%s"
					freshness = "1m"

					schema = "EXTERNAL2"
					table_name = "%s_TABLE_NAME2"
					source_update_table_name = "%s_SOURCE_UPDATE_TABLE_NAME2"
					batch_seq_field = "BATCH_ID"
					is_insert_only = true
					valid_from_field = "TIMESTAMP"

					field {
						name = "BATCH_ID"
						type = "int64"
						sql_type = "NUMBER(38,0)"
						is_hidden = true
					}
					field {
						name = "TIMESTAMP"
						type = "timestamp"
						sql_type = "NUMBER(38,0)"
					}
					field {
						name = "LOG"
						type = "string"
						sql_type = "TEXT"
						is_searchable = "true"
					}
					field {
						name = "TAG"
						type = "variant"
						sql_type = "VARIANT"
					}
					field {
						name = "NEWFIELD"
						type = "object"
						sql_type = "OBJECT"
					}
				}`, randomPrefix+"-rename", randomTablePrefix, randomTablePrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_source_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "name", randomPrefix+"-rename"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "freshness", "1m0s"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "schema", "EXTERNAL2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "table_name", randomTablePrefix+"_TABLE_NAME2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "source_update_table_name", randomTablePrefix+"_SOURCE_UPDATE_TABLE_NAME2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "batch_seq_field", "BATCH_ID"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "is_insert_only", "true"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "valid_from_field", "TIMESTAMP"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_source_dataset.first", "field.*", map[string]string{
						"name":          "BATCH_ID",
						"type":          "int64",
						"sql_type":      "NUMBER(38,0)",
						"is_hidden":     "true",
						"is_enum":       "false",
						"is_searchable": "false",
						"is_const":      "false",
						"is_metric":     "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("observe_source_dataset.first", "field.*", map[string]string{
						"name":          "LOG",
						"type":          "string",
						"sql_type":      "TEXT",
						"is_hidden":     "false",
						"is_enum":       "false",
						"is_searchable": "true",
						"is_const":      "false",
						"is_metric":     "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("observe_source_dataset.first", "field.*", map[string]string{
						"name":      "NEWFIELD",
						"type":      "object",
						"sql_type":  "OBJECT",
						"is_hidden": "false",
					}),
				),
			},
		},
	})
}

func TestInvalidValidFromFieldErrors(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")
	randomTablePrefix := strings.Replace(randomPrefix, "-", "_", -1)

	if os.Getenv("CI") != "true" {
		// The schemas "EXTERNAL" and "EXTERNAL2" were manually created for the provider CI Observe account.
		// While the test still passes even if the schema does not exist, it probably shouldn't, since Snowflake errors on the underlying commands.
		// The tables are entirely non-existent which is likewise not validated.
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's Snowflake database.")
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_source_dataset" "first" {
					workspace = data.observe_workspace.default.oid
					name 	  = "%s"

					schema = "EXTERNAL"
					table_name = "%s_TABLE_NAME"
					source_update_table_name = "%s_SOURCE_UPDATE_TABLE_NAME"
					valid_from_field = "bad_timestamp"
					
					field {
						name = "BATCH_ID"
						type = "int64"
						sql_type = "NUMBER(38,0)"
						is_hidden = true
					}
					field {
						name = "TIMESTAMP"
						type = "timestamp"
						sql_type = "NUMBER(38,0)"
					}
					field {
						name = "LOG"
						type = "string"
						sql_type = "TEXT"
						is_enum = true
						is_searchable = false
						is_hidden = false
						is_const = false
						is_metric = false
					}
					field {
						name = "TAG"
						type = "variant"
						sql_type = "VARIANT"
					}
				}`, randomPrefix, randomTablePrefix, randomTablePrefix),
				ExpectError: regexp.MustCompile(`valid_from_field "bad_timestamp" does not refer to a valid field`),
			},
		},
	})
}
