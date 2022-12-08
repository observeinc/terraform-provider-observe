package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLayeredSettingRecordCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_layered_setting_record" "first" {
					workspace   = data.observe_workspace.default.oid
					name        = "%[1]s"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = observe_datastream.test.dataset
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_layered_setting_record.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_layered_setting_record.first", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting_record.first", "setting", "Scanner.powerLevel"),
				),
			},
		},
	})
}
