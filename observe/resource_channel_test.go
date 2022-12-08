package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// setup a couple of actions and monitors for use with channels
	channelConfigPreamble = configPreamble + datastreamConfigPreamble + `
				resource "observe_monitor" "a" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s/a"

				  inputs = {
				    "test" = observe_datastream.test.dataset
				  }

				  stage {
				    pipeline = "filter true"
				  }

				  rule {
				    count {
				      compare_function   = "greater_or_equal"
				      compare_values     = [100]
					  lookback_time      = "1m"
				    }
				  }
				}
				`
)

func TestAccObserveChannelCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(channelConfigPreamble+`
				resource "observe_channel" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				  icon_url  = "test"
				}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(channelConfigPreamble+`
				resource "observe_channel" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				  icon_url  = "test"
				  monitors = [
				    observe_monitor.a.oid,
				  ]
				}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(channelConfigPreamble+`
				resource "observe_channel" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "0"),
				),
			},
		},
	})
}
