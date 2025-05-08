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
				resource "observe_monitor" "sample_one" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-owengoebel-sample-monitor"
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

				resource "observe_monitor_action" "sample" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-ea"
					icon_url  = "test"

					email {
						target_addresses = [ "test@observeinc.com" ]
					subject_template = "Hello"
					body_template    = "Nope"
					}
				}

				resource "observe_monitor_action_attachment" "sample" {
					action = resource.observe_monitor_action.sample.oid
					monitor = resource.observe_monitor.sample_one.oid
					workspace = data.observe_workspace.default.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.sample", "monitor", "observe_monitor.sample_one", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.sample", "action", "observe_monitor_action.sample", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(monitorActionAttachmentConfigPreamble+`
				resource "observe_monitor" "sample_two" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-owengoebel-sample-monitor"
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

				resource "observe_monitor_action" "sample" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-ea"
					icon_url  = "test"

					email {
						target_addresses = [ "test@observeinc.com" ]
					subject_template = "Hello"
					body_template    = "Nope"
					}
				}

				resource "observe_monitor_action_attachment" "sample" {
					action = resource.observe_monitor_action.sample.oid
					monitor = resource.observe_monitor.sample_two.oid
					workspace = data.observe_workspace.default.oid
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.sample", "monitor", "observe_monitor.sample_two", "oid"),
					resource.TestCheckResourceAttrPair("observe_monitor_action_attachment.sample", "action", "observe_monitor_action.sample", "oid"),
				),
			},
		},
	})
}
