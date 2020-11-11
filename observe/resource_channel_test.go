package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// setup a couple of actions for use in channels
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
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel.example", "actions.#", "2"),
				),
			},
		},
	})
}
