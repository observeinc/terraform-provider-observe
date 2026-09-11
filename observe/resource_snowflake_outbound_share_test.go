package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccObserveSnowflakeOutboundShare(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	renamedPrefix := randomPrefix + "-renamed"
	var originalID string

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
					func(s *terraform.State) error {
						originalID = s.RootModule().Resources["observe_snowflake_outbound_share.test"].Primary.ID
						return nil
					},
				),
			},
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
				`, renamedPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_snowflake_outbound_share.test", "name", renamedPrefix),
					func(s *terraform.State) error {
						newID := s.RootModule().Resources["observe_snowflake_outbound_share.test"].Primary.ID
						if newID == originalID {
							return fmt.Errorf("expected snowflake outbound share ID to change on rename, got %s", newID)
						}
						return nil
					},
				),
			},
		},
	})
}
