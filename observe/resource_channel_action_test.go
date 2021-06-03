package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveChannelActionCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_channel_action" "action" {
				  workspace = data.observe_workspace.kubernetes.oid
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
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.url", "https://example.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.body", "{}"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.headers.test", "hello"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_channel_action" "action" {
				  workspace  = data.observe_workspace.kubernetes.oid
				  name       = "%s"
				  icon_url   = "test"

				  webhook {
				  	url 	= "https://observeinc.com"
					body 	= "nope"
				  }
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "1m"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.url", "https://observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.body", "nope"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "webhook.0.headers.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_channel_action" "action" {
				  workspace  = data.observe_workspace.kubernetes.oid
				  name       = "%s"
				  icon_url   = "test"
				  rate_limit = "5m"

				  email {
				  	to 		= [ "test@observeinc.com" ]
					subject = "Hello"
					body 	= "Nope"
				  }
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "5m"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.#", "1"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.subject", "Hello"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.body", "Nope"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_channel_action" "action" {
				  workspace = data.observe_workspace.kubernetes.oid
				  name      = "%s"
				  icon_url  = "filing-cabinet"

				  email {
				  	to 		= [ "debug@observeinc.com", "test@observeinc.com" ]
					subject = "Nope"
					body 	= "Hello"
					is_html = true
				  }
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_channel_action.action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_channel_action.action", "icon_url", "filing-cabinet"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "rate_limit", "1m"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.#", "2"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.0", "debug@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.to.1", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.subject", "Nope"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.body", "Hello"),
					resource.TestCheckResourceAttr("observe_channel_action.action", "email.0.is_html", "true"),
				),
			},
		},
	})
}
