package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveMonitorV2ActionEmailDatasource(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					data "observe_user" "system" {
						email = "%[2]s"
					}

					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "somebody once told me"
							body = "the world is gonna roll me"
							fragments = jsonencode({
								foo = "bar"
							})
							addresses = ["test@observeinc.com"]
							users = [data.observe_user.system.oid]
						}
						name = "%[1]s"
						description = "an interesting description"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "type", "email"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.subject", "somebody once told me"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.body", "the world is gonna roll me"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.users.#", "1"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2ActionWebhookDatasource(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "webhook"
						webhook {
							headers {
								header = "never gonna give you up"
								value = "never gonna let you down"
							}
							body = "never gonna run around and desert you"
							fragments = jsonencode({
								foo = "bar"
							})
							url = "https://example.com/"
							method = "post"
						}
						name = "%[1]s"
						description = "an interesting description"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "type", "webhook"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.headers.0.header", "never gonna give you up"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.headers.0.value", "never gonna let you down"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.body", "never gonna run around and desert you"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.url", "https://example.com/"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.method", "post"),
				),
			},
		},
	})
}
