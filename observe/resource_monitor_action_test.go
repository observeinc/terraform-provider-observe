package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveMonitorAction_Webhook(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor_action" "webhook_action" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					icon_url  = "test"

					webhook {
						url_template 	= "https://example.com"
						body_template 	= "{}"
						headers 		= {
							"test" = "hello"
						}
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_action.webhook_action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_action.webhook_action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_monitor_action.webhook_action", "webhook.0.url_template", "https://example.com"),
					resource.TestCheckResourceAttr("observe_monitor_action.webhook_action", "webhook.0.body_template", "{}"),
					resource.TestCheckResourceAttr("observe_monitor_action.webhook_action", "webhook.0.headers.test", "hello"),
				),
			},
		},
	})
}

func TestAccObserveMonitorAction_Email(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor_action" "email_action" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					icon_url  = "test"

					email {
						target_addresses = [ "test@observeinc.com" ]
					subject_template = "Hello"
					body_template    = "Nope"
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "email.0.target_addresses.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "email.0.target_addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "email.0.subject_template", "Hello"),
					resource.TestCheckResourceAttr("observe_monitor_action.email_action", "email.0.body_template", "Nope"),
				),
			},
		},
	})
}
