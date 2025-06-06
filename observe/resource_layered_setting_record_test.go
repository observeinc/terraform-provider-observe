package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLayeredSettingRecord(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_layered_setting_record" "datasource_int64" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-dataset-int64"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = observe_datastream.test.dataset
				}
				
				resource "observe_layered_setting_record" "datastream_int64" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-datastream-int64"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = observe_datastream.test.oid
				}

				resource "observe_layered_setting_record" "datasource_bool" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-dataset-bool"
					setting     = "Dataset.periodicReclusteringDisabled"
					value_bool  = false
					target      = observe_datastream.test.dataset
				}
				
				resource "observe_rbac_group" "limit_power" {
					name = "%[1]s-limit_power"
				}
				resource "observe_layered_setting_record" "group_int64" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s-group-int64"
					setting     = "Scanner.powerLevel"
					value_int64 = 75
					target      = observe_rbac_group.limit_power.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_int64", "name", randomPrefix+"-dataset-int64"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_int64", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_int64", "setting", "Scanner.powerLevel"),

					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream_int64", "name", randomPrefix+"-datastream-int64"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream_int64", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream_int64", "setting", "Scanner.powerLevel"),

					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_bool", "name", randomPrefix+"-dataset-bool"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_bool", "value_bool", "false"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource_bool", "setting", "Dataset.periodicReclusteringDisabled"),

					resource.TestCheckResourceAttr("observe_layered_setting_record.group_int64", "name", randomPrefix+"-group-int64"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.group_int64", "value_int64", "75"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.group_int64", "setting", "Scanner.powerLevel"),
					resource.TestCheckResourceAttrSet("observe_layered_setting_record.group_int64", "target"),
				),
			},
		},
	})
}
