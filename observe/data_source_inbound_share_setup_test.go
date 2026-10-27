package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveInboundShareSetup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `data "observe_inbound_share_setup" "setup" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "consumer_account"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "region"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "setup_sql.0.sql"),
				),
			},
		},
	})
}
