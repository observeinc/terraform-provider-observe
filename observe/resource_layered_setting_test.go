package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLayeredSettingCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_layered_setting" "first" {
					workspace   = data.observe_workspace.default.oid
					name        = "%s"
					setting     = "Scanner.powerLevel"
					value_int64 = 9009
					target      = data.observe_dataset.observation.oid
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_layered_setting.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_layered_setting.first", "value_int64", "9009"),
					resource.TestCheckResourceAttr("observe_layered_setting.first", "setting", "Scanner.powerLevel"),
				),
			},
		},
	})
}
