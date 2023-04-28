package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLayeredSettingRecord(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_layered_setting_record" "datasource" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = observe_datastream.test.dataset
				}
				
				resource "observe_layered_setting_record" "datastream" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = observe_datastream.test.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datasource", "setting", "Scanner.powerLevel"),

					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.datastream", "setting", "Scanner.powerLevel"),
				),
			},
		},
	})
}
