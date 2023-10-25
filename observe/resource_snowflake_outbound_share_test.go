package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSnowflakeShareOutbound(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_snowflake_outbound_share" "test" {
						workspace   = data.observe_workspace.default.oid
						name        = "%[1]s"
						description = "test description"

						account {
							account = "io79077"
							organization = "HC83707"
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_snowflake_outbound_share.test", "workspace"),
					resource.TestCheckResourceAttrSet("observe_snowflake_outbound_share.test", "oid"),
					resource.TestCheckResourceAttr("observe_snowflake_outbound_share.test", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_snowflake_outbound_share.test", "description", "test description"),
					resource.TestCheckResourceAttrSet("observe_snowflake_outbound_share.test", "share_name"),
				),
			},
		},
	})
}
