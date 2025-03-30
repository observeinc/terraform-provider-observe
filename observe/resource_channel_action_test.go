package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// setup a couple of actions and monitors for use with channels
	channelActionConfigPreamble = configPreamble + `
				resource "observe_channel" "a" {
					workspace = data.observe_workspace.default.oid
					name      = "%s/a"
					icon_url  = "test"
				}

				resource "observe_channel" "b" {
					workspace = data.observe_workspace.default.oid
					name      = "%s/b"
					icon_url  = "test"
				}
				`
)

func TestAccObserveChannelActionCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(channelActionConfigPreamble+`
				resource "observe_channel_action" "action" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					icon_url  = "test"

					webhook {
						url 	= "https://example.com"
						body 	= "{}"
						headers = {
							"test" = "hello"
						}
					}
				}
				`, randomPrefix, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "channels.#", "0"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.url", "https://example.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.body", "{}"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.headers.test", "hello"),
				),
			},
			{
				Config: fmt.Sprintf(channelActionConfigPreamble+`
				resource "observe_channel_action" "action" {
					workspace  = data.observe_workspace.default.oid
					name       = "%s"
					icon_url   = "test"

					webhook {
						url 	= "https://observeinc.com"
						body 	= "nope"
					}
				}
				`, randomPrefix, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "channels.#", "0"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "10m0s"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.url", "https://observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.body", "nope"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.headers.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(channelActionConfigPreamble+`
				resource "observe_channel_action" "action" {
					workspace  = data.observe_workspace.default.oid
					name       = "%s"
					icon_url   = "test"
					rate_limit = "11m"

					notify_on_close = true

					channels  = [
						observe_channel.a.oid,
					]

					email {
						to 		= [ "test@observeinc.com" ]
						subject = "Hello"
						body 	= "Nope"
					}
				}
				`, randomPrefix, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "11m0s"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "channels.#", "1"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.#", "1"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.subject", "Hello"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.body", "Nope"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "notify_on_close", "true"),
				),
			},
			{
				Config: fmt.Sprintf(channelActionConfigPreamble+`
				resource "observe_channel_action" "action" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					icon_url  = "filing-cabinet"
					rate_limit = "10m"
					channels  = [
						observe_channel.a.oid,
						observe_channel.b.oid,
					]

					email {
						to 		= [ "debug@observeinc.com", "test@observeinc.com" ]
						subject = "Nope"
						body 	= "Hello"
						is_html = true
					}
				}
				`, randomPrefix, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "filing-cabinet"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "channels.#", "2"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "10m0s"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.#", "2"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.0", "debug@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.1", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.subject", "Nope"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.body", "Hello"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.is_html", "true"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "notify_on_close", "false"),
				),
			},
		},
	})
}
