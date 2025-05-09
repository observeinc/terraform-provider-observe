package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var monitorV2ConfigPreamble = configPreamble + datastreamConfigPreamble

func TestAccObserveMonitorV2Count(t *testing.T) {
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
								freshness_goal= "15m"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "count"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.count.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.transform.0.freshness_goal", "15m0s"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2Threshold(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "threshold"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14, groupme:12"
						}
						groupings {
							column_path {
								name = "groupme"
							}
						}
						no_data_rules {
							expiration = "30m"
							threshold {
								value_column_name = "temp_number"
								aggregation = "all_of"
							}
						}
						rules {
							level = "informational"
							threshold {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
								value_column_name = "temp_number"
								aggregation = "all_of"
								compare_groups {
									column {
										column_path {
											name = "groupme"
										}
									}
									compare_values {
										compare_fn = "not_equal"
										value_int64 = [12]
									}
								}
							}
						}
						scheduling {
							transform {
								freshness_goal = "15m"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "threshold"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "no_data_rules.0.expiration", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "no_data_rules.0.threshold.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "no_data_rules.0.threshold.0.aggregation", "all_of"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.aggregation", "all_of"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.compare_groups.0.column.0.column_path.0.name", "groupme"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.compare_groups.0.compare_values.0.compare_fn", "not_equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.threshold.0.compare_groups.0.compare_values.0.value_int64.0", "12"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.transform.0.freshness_goal", "15m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "groupings.0.column_path.0.name", "groupme"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2Promote(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "promote"
						name = "%[1]s"
						lookback_time = "0s"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = <<-EOF
								colmake temp_number:14
								colmake temp_string:"test"
								colmake temp_duration:5m
							EOF
						}
						rules {
							level = "informational"
							promote {
								compare_columns {
									compare_values {
										compare_fn = "greater"
										value_int64 = [1]
									}
									column {
										column_path {
											name = "temp_number"
										}
									}
								}
							}
						}
						rules {
							level = "error"
							promote {
								compare_columns {
									compare_values {
										compare_fn = "not_contains"
										value_string = ["test"]
									}
									column {
										column_path {
											name = "temp_string"
										}
									}
								}
							}
						}
						scheduling {
							transform {
								freshness_goal= "15m"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "promote"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.0.value_int64.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.column.0.column_path.0.name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.level", "error"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.promote.0.compare_columns.0.compare_values.0.compare_fn", "not_contains"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.promote.0.compare_columns.0.compare_values.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.promote.0.compare_columns.0.column.0.column_path.0.name", "temp_string"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.transform.0.freshness_goal", "15m0s"),
				),
			},
			// now test update
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "promote"
						name = "%[1]s"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = <<-EOF
								colmake temp_number:14
								colmake temp_string:"test"
								colmake temp_duration:5m
							EOF
						}
						rules {
							level = "informational"
							promote {
								compare_columns {
									compare_values {
										compare_fn = "greater"
										value_duration = ["1m"]
									}
									column {
										column_path {
											name = "temp_duration"
										}
									}
								}
							}
						}
						rules {
							level = "error"
							promote {
								compare_columns {
									compare_values {
										compare_fn = "not_contains"
										value_string = ["test"]
									}
									column {
										column_path {
											name = "temp_string"
										}
									}
								}
							}
						}
						scheduling {
							transform {
								freshness_goal= "15m"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.0.value_duration.0", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.column.0.column_path.0.name", "temp_duration"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2MultipleActionsViaOneShot(t *testing.T) {
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
								freshness_goal = "15m"
							}
						}
						max_alerts_per_hour = 99
						actions {
							action {
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
								description = "an interesting description 1"
							}
							levels = ["informational"]
							send_end_notifications = true
							send_reminders_interval = "10m"
						}
						actions {
							action {
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
								description = "an interesting description 2"
							}
							levels = ["informational"]
							conditions {
								compare_terms {
									comparison {
										compare_fn = "equal"
										value_string = ["test"]
									}
									column {
										column_path  {
											name = "description"
										}
									}
								}
							}
							send_end_notifications = false
							send_reminders_interval = "20m"
						}
						actions {
							action {
								type = "webhook"
								webhook {
									url = "https://example.com"
									method = "post"
									body = "test"
								}
								description = "an interesting description 3"
							}
							levels = ["error"]
							send_end_notifications = false
							send_reminders_interval = "30m"
						}
					}

					data "observe_user" "system" {
						email = "%[2]s"
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "max_alerts_per_hour", "99"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.description", "an interesting description 1"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.send_reminders_interval", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.description", "an interesting description 2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.send_reminders_interval", "20m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.comparison.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.column.0.column_path.0.name", "description"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.type", "webhook"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.description", "an interesting description 3"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.webhook.0.url", "https://example.com"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.webhook.0.method", "post"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.webhook.0.body", "test"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.send_reminders_interval", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.transform.0.freshness_goal", "15m0s"),
				),
			},
			// now test update
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[1]s"
						lookback_time = "15m"
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
								freshness_goal = "15m"
							}
						}
						custom_variables = jsonencode({
							fizz = "buzz"
						})
						max_alerts_per_hour = 99
						actions {
							action {
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
								description = "an interesting description 1"
							}
							levels = ["informational"]
							send_end_notifications = true
							send_reminders_interval = "11m"
						}
						actions {
							action {
								type = "webhook"
								webhook {
									url = "https://example.com"
									method = "post"
									body = "test"
								}
								description = "an interesting description 3 - reordered"
							}
							levels = ["error"]
							send_end_notifications = false
							send_reminders_interval = "33m"
						}
						actions {
							action {
								type = "email"
								email {
									subject = "never gonna give you up"
									body = "never gonna let you down"
									fragments = jsonencode({
										fizz = "boo"
									})
									addresses = ["test@observeinc.com"]
									users = [data.observe_user.system.oid]
								}
								description = "an interesting description 2"
							}
							levels = ["informational"]
							conditions {
								compare_terms {
									comparison {
										compare_fn = "equal"
										value_string = ["test"] 
									}
									column {
										column_path  {
											name = "description"
										}
									}
								}
							}
							send_end_notifications = false
							send_reminders_interval = "22m"
						}
					}

					data "observe_user" "system" {
						email = "%[2]s"
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "15m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.description", "an interesting description 1"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.send_reminders_interval", "11m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.type", "webhook"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.description", "an interesting description 3 - reordered"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.send_reminders_interval", "33m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.description", "an interesting description 2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.send_reminders_interval", "22m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "custom_variables", `{"fizz":"buzz"}`),
				),
			},
		},
	})
}

func TestAccObserveMonitorIntervals(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "threshold"
						name = "%[1]s"
						lookback_time = "10m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14, groupme:12"
						}
						rules {
							level = "informational"
							threshold {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
								value_column_name = "temp_number"
								aggregation = "all_of"
							}
						}
						scheduling {
							interval {
								interval = "5m"
								randomize = "1m"
							}
						}
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.interval", "5m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.interval.0.randomize", "1m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.interval", "5m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.randomize", "1m0s"),
				),
			},
		},
	})
}

func TestAccObserveMonitorRawCron(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "threshold"
						name = "%[1]s"
						lookback_time = "10m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14, groupme:12"
						}
						rules {
							level = "informational"
							threshold {
								compare_values {
									compare_fn = "greater"
									value_int64 = [0]
								}
								value_column_name = "temp_number"
								aggregation = "all_of"
							}
						}
						scheduling {
							scheduled {
								raw_cron = "0 0 * * *"
								timezone = "America/New_York"
							}
						}
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.raw_cron", "0 0 * * *"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.timezone", "America/New_York"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.raw_cron", "0 0 * * *"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.timezone", "America/New_York"),
				),
			},
		},
	})
}
