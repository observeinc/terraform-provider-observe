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
						comment = "a descriptive comment"
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
							interval {
								interval = "15m"
								randomize = "0"
							}
						}
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
						}
						name = "%[1]s"
						description = "an interesting description"
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.subject", "somebody once told me"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "email.0.body", "the world is gonna roll me"),
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
						comment = "a descriptive comment"
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
							interval {
								interval = "15m"
								randomize = "0"
							}
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
						}
						name = "%[1]s"
						description = "an interesting description"
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "type", "webhook"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.headers.0.header", "never gonna give you up"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.headers.0.value", "never gonna let you down"),
					resource.TestCheckResourceAttr("observe_monitor_v2_action.act", "webhook.0.body", "never gonna run around and desert you"),
				),
			},
		},
	})
}
