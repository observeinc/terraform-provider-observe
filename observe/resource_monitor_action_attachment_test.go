package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	monitorActionAttachmentConfigPreamble = monitorConfigPreamble + `
				resource "observe_monitor" "first" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s-first"
				  freshness = "4m"

				  comment = "a descriptive comment"

				  inputs = {
				  	"test" = observe_datastream.test.dataset
				  }

				  stage {}

				  rule {
					count {
					  compare_function   = "less_or_equal"
					  compare_values     = [1]
					  lookback_time      = "1m"
					}
				  }
				}

				resource "observe_monitor" "second" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s-second"
				  freshness = "4m"

				  comment = "a descriptive comment"

				  inputs = {
				  	"test" = observe_datastream.test.dataset
				  }

				  stage {}

				  rule {
					count {
					  compare_function   = "less_or_equal"
					  compare_values     = [1]
					  lookback_time      = "1m"
					}
				  }
				}

				resource "observe_monitor_action" "webhook_action" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s-wa"
				  icon_url  = "test"

				  webhook {
				  	url_template 	= "https://example.com"
					body_template 	= "{}"
					headers 		= {
						"test" = "hello"
					}
				  }
				}
				
				resource "observe_monitor_action" "email_action" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s-ea"
				  icon_url  = "test"

				  email {
				  	target_addresses = [ "test@observeinc.com" ]
					subject_template = "Hello"
					body_template    = "Nope"
				  }
				}
				`
)

func TestAccObserveMonitorActionAttachment_OneToOne(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "one_to_one" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.first.oid
				  action = observe_monitor_action.webhook_action.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "monitor", "observe_monitor.first", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "action", "observe_monitor_action.webhook_action", "oid"),
				),
			},
		},
	})
}

func TestAccObserveMonitorActionAttachment_Named(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "one_to_one" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.first.oid
				  action = observe_monitor_action.webhook_action.oid
				  name = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_action_attachment.one_to_one", "name", randomPrefix),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "monitor", "observe_monitor.first", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "action", "observe_monitor_action.webhook_action", "oid"),
				),
			},
		},
	})
}

func TestAccObserveMonitorActionAttachment_ManyToMany(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "first_and_webhook" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.first.oid
				  action = observe_monitor_action.webhook_action.oid
				}
				
				resource "observe_monitor_action_attachment" "first_and_email" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.first.oid
				  action = observe_monitor_action.email_action.oid
				}
				
				resource "observe_monitor_action_attachment" "second_and_webhook" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.second.oid
				  action = observe_monitor_action.webhook_action.oid
				}
				
				resource "observe_monitor_action_attachment" "second_and_email" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.second.oid
				  action = observe_monitor_action.email_action.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.first_and_webhook", "monitor", "observe_monitor.first", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.first_and_webhook", "action", "observe_monitor_action.webhook_action", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.first_and_email", "monitor", "observe_monitor.first", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.first_and_email", "action", "observe_monitor_action.email_action", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.second_and_webhook", "monitor", "observe_monitor.second", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.second_and_webhook", "action", "observe_monitor_action.webhook_action", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.second_and_email", "monitor", "observe_monitor.second", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.second_and_email", "action", "observe_monitor_action.email_action", "oid"),
				),
			},
		},
	})
}

func TestAccObserveMonitorActionAttachment_UpdateMonitorActionAttachment(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "one_to_one" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.first.oid
				  action = observe_monitor_action.webhook_action.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "monitor", "observe_monitor.first", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "action", "observe_monitor_action.webhook_action", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "one_to_one" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.second.oid
				  action = observe_monitor_action.webhook_action.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "monitor", "observe_monitor.second", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "action", "observe_monitor_action.webhook_action", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor_action_attachment" "one_to_one" {
				  workspace = data.observe_workspace.default.oid
				  monitor = observe_monitor.second.oid
				  action = observe_monitor_action.email_action.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "monitor", "observe_monitor.second", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.one_to_one", "action", "observe_monitor_action.email_action", "oid"),
				),
			},
		},
	})
}

func TestAccObserveMonitorActionAttachment_ChangeMonitorResourceName(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				data "observe_workspace" "default" {
				name = "Default"
				}

				data "observe_dataset" "default" {
				workspace = data.observe_workspace.default.oid
				name      = "Default"
				}

				resource "observe_monitor" "battery_level_is_low" {
					definition  = jsonencode({})
					disabled    = false
					inputs      = {
						"battery" = data.observe_dataset.default.oid
					}
					is_template = false
					name        = "vikram/Battery level is very low"
					workspace   = data.observe_workspace.default.oid

					notification_spec {
						importance         = "informational"
						merge              = "separate"
						notify_on_close    = false
					}

					rule {
						promote {
							description_field = "BUNDLE_TIMESTAMP"
							kind_field        = "FIELDS"
							primary_key       = [
								"BUNDLE_ID",
							]
						}
					}

					stage {
						pipeline = <<-EOF
							filter DATASTREAM_ID = "4f7fc854-53ae-4ace-8530-906417001"
						EOF
						output_stage = true
					}
				}

				resource "observe_monitor_action" "test" {
					name = "test"
					workspace = data.observe_workspace.default.oid
					description = "test"
					email {
						body_template = "./slack.tpl"
						subject_template = "test"
						target_addresses = ["vikram@observeinc.com"]
					}
				}

				resource "observe_monitor_action_attachment" "test" {
					action = resource.observe_monitor_action.test.oid
					monitor = resource.observe_monitor.battery_level_is_low.oid
					workspace = data.observe_workspace.default.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.test", "monitor", "observe_monitor.battery_level_is_low", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.test", "action", "observe_monitor_action.test", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				data "observe_workspace" "default" {
				name = "Default"
				}

				data "observe_dataset" "default" {
				workspace = data.observe_workspace.default.oid
				name      = "Default"
				}

				resource "observe_monitor" "battery_level_is_very_low" {
					definition  = jsonencode({})
					disabled    = false
					inputs      = {
						"battery" = data.observe_dataset.default.oid
					}
					is_template = false
					name        = "vikram/Battery level is very low"
					workspace   = data.observe_workspace.default.oid

					notification_spec {
						importance         = "informational"
						merge              = "separate"
						notify_on_close    = false
					}

					rule {
						promote {
							description_field = "BUNDLE_TIMESTAMP"
							kind_field        = "FIELDS"
							primary_key       = [
								"BUNDLE_ID",
							]
						}
					}

					stage {
						pipeline = <<-EOF
							filter DATASTREAM_ID = "4f7fc854-53ae-4ace-8530-906417001"
						EOF
						output_stage = true
					}
				}

				resource "observe_monitor_action" "test" {
					name = "test"
					workspace = data.observe_workspace.default.oid
					description = "test"
					email {
						body_template = "./slack.tpl"
						subject_template = "test"
						target_addresses = ["vikram@observeinc.com"]
					}
				}

				resource "observe_monitor_action_attachment" "test" {
					action = resource.observe_monitor_action.test.oid
					monitor = resource.observe_monitor.battery_level_is_very_low.oid
					workspace = data.observe_workspace.default.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.test", "monitor", "observe_monitor.battery_level_is_very_low", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.test", "action", "observe_monitor_action.test", "oid"),
				),
			},
		},
	})
}
