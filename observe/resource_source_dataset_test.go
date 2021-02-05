package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDatasetResource(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_source_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"

					schema = "EXTERNAL"
					table_name = "TABLE_NAME"
					source_update_table_name = "SOURCE_UPDATE_TABLE_NAME"
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
						is_searchable = true
						is_hidden = false
						is_const = false
						is_metric = false
					}
					field {
						name = "TAG"
						type = "any"
						sql_type = "VARIANT"
					}

				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_source_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "schema", "EXTERNAL"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "table_name", "TABLE_NAME"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "source_update_table_name", "SOURCE_UPDATE_TABLE_NAME"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "valid_from_field", "TIMESTAMP"),
					resource.TestCheckNoResourceAttr("observe_source_dataset.first", "freshness"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.name", "BATCH_ID"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.type", "int64"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.sql_type", "NUMBER(38,0)"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.is_hidden", "true"),
				),
			},
			{
				Config: configPreamble + fmt.Sprintf(`
				resource "observe_source_dataset" "first" {
					workspace = data.observe_workspace.kubernetes.oid
					name 	  = "%s"
					freshness = "1m"

					schema = "EXTERNAL2"
					table_name = "TABLE_NAME2"
					source_update_table_name = "SOURCE_UPDATE_TABLE_NAME2"
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
						type = "any"
						sql_type = "VARIANT"
					}
					field {
						name = "NEWFIELD"
						type = "object"
						sql_type = "OBJECT"
					}
				}`, randomPrefix+"-rename"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_source_dataset.first", "workspace"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "name", randomPrefix+"-rename"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "freshness", "1m0s"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "schema", "EXTERNAL2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "table_name", "TABLE_NAME2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "source_update_table_name", "SOURCE_UPDATE_TABLE_NAME2"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "batch_seq_field", "BATCH_ID"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "is_insert_only", "true"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "valid_from_field", "TIMESTAMP"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.name", "BATCH_ID"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.type", "int64"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.sql_type", "NUMBER(38,0)"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.0.is_hidden", "true"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.4.name", "NEWFIELD"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.4.type", "object"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.4.sql_type", "OBJECT"),
					resource.TestCheckResourceAttr("observe_source_dataset.first", "field.4.is_hidden", "false"),
				),
			},
		},
	})
}