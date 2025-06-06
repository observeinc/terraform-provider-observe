package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceMonitorAction_Webhook(t *testing.T) {
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

				data "observe_monitor_action" "lookup" {
					id     = observe_monitor_action.webhook_action.id
				}

				data "observe_monitor_action" "lookup_by_name" {
					workspace = data.observe_workspace.default.oid
					name      = observe_monitor_action.webhook_action.name
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "icon_url", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "webhook.0.url_template", "https://example.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "webhook.0.body_template", "{}"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "webhook.0.headers.test", "hello"),

					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "icon_url", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "webhook.0.url_template", "https://example.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "webhook.0.body_template", "{}"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "webhook.0.headers.test", "hello"),
				),
			},
		},
	})
}

func TestAccObserveSourceMonitorAction_Email(t *testing.T) {
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

				data "observe_monitor_action" "lookup" {
					id     = observe_monitor_action.email_action.id
				}

				data "observe_monitor_action" "lookup_by_name" {
					workspace = data.observe_workspace.default.oid
					name      = observe_monitor_action.email_action.name
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "icon_url", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "email.0.target_addresses.#", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "email.0.target_addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "email.0.subject_template", "Hello"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup", "email.0.body_template", "Nope"),

					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "icon_url", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "email.0.target_addresses.#", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "email.0.target_addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "email.0.subject_template", "Hello"),
					resource.TestCheckResourceAttr("data.observe_monitor_action.lookup_by_name", "email.0.body_template", "Nope"),
				),
			},
		},
	})
}
