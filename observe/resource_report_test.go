package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveReport(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix1 := acctest.RandomWithPrefix("tf")
	randomPrefix2 := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		Providers:  testAccProviders,
		IsUnitTest: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble+`
					resource "observe_report" "first" {
						label = "%s"
						enabled = true
						dashboard {
							id = observe_dashboard.first.id
							parameters {
								key = "test"
								value = "testvalue"
							}
							query_window_duration_minutes = 10
						}
						schedule {
							frequency = "Weekly"
							every = 1
							time_of_day = "12:00"
							timezone = "UTC"
							day_of_the_week = "Monday"
						}
						email_subject = "test"
						email_recipients = ["test@example.com"]
						email_body = "test"
					}
				`, randomPrefix1, randomPrefix2, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_report.first", "label", randomPrefix2),
					resource.TestCheckResourceAttr("observe_report.first", "enabled", "true"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.parameters.0.key", "test"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.parameters.0.value", "testvalue"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.query_window_duration_minutes", "10"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.frequency", "Weekly"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.every", "1"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.time_of_day", "12:00"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.timezone", "UTC"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.day_of_the_week", "Monday"),
					resource.TestCheckResourceAttr("observe_report.first", "email_subject", "test"),
					resource.TestCheckResourceAttr("observe_report.first", "email_recipients.#", "1"),
					resource.TestCheckResourceAttr("observe_report.first", "email_recipients.0", "test@example.com"),
					resource.TestCheckResourceAttr("observe_report.first", "email_body", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble+`
					resource "observe_report" "first" {
						label = "%s-NEW"
						enabled = true
						dashboard {
							id = observe_dashboard.first.id
							parameters {
								key = "test2"
								value = "testvalue2"
							}
							query_window_duration_minutes = 100
						}
						schedule {
							frequency = "Monthly"
							every = 3
							time_of_day = "13:37"
							timezone = "UTC"
							day_of_the_month = 15
						}
						email_subject = "test-updated"
						email_recipients = ["test-updated@example.com", "test-updated2@example.com"]
						email_body = "test-updated"
					}
				`, randomPrefix1, randomPrefix2, randomPrefix2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_report.first", "label", randomPrefix2+"-NEW"),
					resource.TestCheckResourceAttr("observe_report.first", "enabled", "true"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.parameters.0.key", "test2"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.parameters.0.value", "testvalue2"),
					resource.TestCheckResourceAttr("observe_report.first", "dashboard.0.query_window_duration_minutes", "100"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.frequency", "Monthly"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.every", "3"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.time_of_day", "13:37"),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.timezone", "UTC"),
					// day_of_the_week should be empty because it's not used for Monthly
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.day_of_the_week", ""),
					resource.TestCheckResourceAttr("observe_report.first", "schedule.0.day_of_the_month", "15"),
					resource.TestCheckResourceAttr("observe_report.first", "email_subject", "test-updated"),
					resource.TestCheckResourceAttr("observe_report.first", "email_recipients.#", "2"),
					resource.TestCheckResourceAttr("observe_report.first", "email_recipients.0", "test-updated@example.com"),
					resource.TestCheckResourceAttr("observe_report.first", "email_recipients.1", "test-updated2@example.com"),
					resource.TestCheckResourceAttr("observe_report.first", "email_body", "test-updated"),
				),
			},
		},
	})
}
