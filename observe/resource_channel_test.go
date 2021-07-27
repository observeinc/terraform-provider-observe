package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// setup a couple of actions and monitors for use with channels
	channelConfigPreamble = configPreamble + `
				resource "observe_channel_action" "a" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "a"

				  webhook {
				    url 	= "https://example.com"
					body 	= "{}"
				  }
				}

				resource "observe_channel_action" "b" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "b"

				  email {
				    to      = ["test@observeinc.com"]
					subject = "Test"
					body 	= "Test"
				  }
				}

				resource "observe_monitor" "a" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "a"

				  inputs = {
				    "observation" = data.observe_dataset.observation.oid
				  }

				  stage {
				    pipeline = "filter true"
				  }

				  rule {
				    group_by      = "none"

				    count {
				      compare_function   = "greater_or_equal"
				      compare_values     = [100]
					  lookback_time      = "1m"
				    }
				  }

				  notification_spec {
				    selection       = "count"
				    selection_value = 1
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
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "%s"
				  icon_url  = "test"
				  actions   = [
				    observe_channel_action.a.oid,
				  ]
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "actions.#", "1"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(channelConfigPreamble+`
				resource "observe_channel" "example" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "%s"
				  icon_url  = "test"
				  actions   = [
				    observe_channel_action.b.oid,
					observe_channel_action.a.oid,
				  ]
				  monitors = [
				    observe_monitor.a.oid,
				  ]
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "actions.#", "2"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(channelConfigPreamble+`
				resource "observe_channel" "example" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "%s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "actions.#", "0"),
					resource.TestCheckResourceAttr("observe_channel.example", "monitors.#", "0"),
				),
			},
		},
	})
}
