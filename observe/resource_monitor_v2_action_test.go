package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveMonitorV2ActionEmail(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = <<-EOF
								colmake kind:"test", description:"test"
							EOF
							output_stage = true
						}
						stage {
							pipeline = <<-EOF
								filter kind ~ "test"
							EOF
						}
						rules {
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
							}
						}
						scheduling {
							transform {
								freshness_target = "15m"
							}
						}
						actions {
							oid = observe_monitor_v2_action.act.oid
						}
					}

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
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.#", "1"),
					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.subject", "somebody once told me"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.body", "the world is gonna roll me"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.users.#", "1"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2ActionWebhook(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = <<-EOF
								colmake kind:"test", description:"test"
							EOF
							output_stage = true
						}
						stage {
							pipeline = <<-EOF
								filter kind ~ "test"
							EOF
						}
						rules {
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
							}
						}
						scheduling {
							transform {
								freshness_target = "15m"
							}
						}
						actions {
							oid = observe_monitor_v2_action.act.oid
						}
					}

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
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.#", "1"),
					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "type", "webhook"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.headers.0.header", "never gonna give you up"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.headers.0.value", "never gonna let you down"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.body", "never gonna run around and desert you"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.url", "https://example.com/"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.method", "post"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2MultipleActionsEmail(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = <<-EOF
								colmake kind:"test", description:"test"
							EOF
							output_stage = true
						}
						stage {
							pipeline = <<-EOF
								filter kind ~ "test"
							EOF
						}
						rules {
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
							}
						}
						scheduling {
							transform {
								freshness_target = "15m"
							}
						}
						actions {
							oid = observe_monitor_v2_action.act1.oid
							levels = ["informational"]
							send_end_notifications = true
							send_reminders_interval = "10m"
						}
						actions {
							oid = observe_monitor_v2_action.act2.oid
							levels = ["informational"]
							send_end_notifications = false
							send_reminders_interval = "20m"
						}
					}

					data "observe_user" "system" {
						email = "%[2]s"
					}

					resource "observe_monitor_v2_action" "act1" {
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
						name = "%[1]s-1"
						description = "an interesting description 1"
					}

					resource "observe_monitor_v2_action" "act2" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "never gonna give you up"
							body = "never gonna let you down"
							fragments = jsonencode({
								fizz = "buzz"
							})
							addresses = ["test@observeinc.com"]
							users = [data.observe_user.system.oid]
						}
						name = "%[1]s-2"
						description = "an interesting description 2"
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.#", "2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.levels.0", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.send_end_notifications", "true"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.send_reminders_interval", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.levels.0", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.send_end_notifications", "false"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.send_reminders_interval", "20m0s"),

					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act1", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "description", "an interesting description 1"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "email.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "email.0.subject", "somebody once told me"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "email.0.body", "the world is gonna roll me"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "email.0.addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act1", "email.0.users.#", "1"),

					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act2", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "name", randomPrefix+"-2"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "description", "an interesting description 2"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "email.0.fragments", "{\"fizz\":\"buzz\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "email.0.subject", "never gonna give you up"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "email.0.body", "never gonna let you down"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "email.0.addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act2", "email.0.users.#", "1"),
				),
			},
		},
	})
}
