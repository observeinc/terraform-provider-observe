package observe

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
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
					// -1 is the sentinel value for null, see comments in resource definition
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "max_alerts_per_hour", "-1"),
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
								compare_terms {
									comparison {
										compare_fn = "equal"
										value_string = ["test"]
									}
									column {
										column_path  {
											name = "kind"
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
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.operator", "and"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.comparison.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.0.column.0.column_path.0.name", "description"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.1.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.1.comparison.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.conditions.0.compare_terms.1.column.0.column_path.0.name", "kind"),
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
						max_alerts_per_hour = 0
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
							levels = ["no_data"]
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
								operator = "or"
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
								compare_terms {
									comparison {
										compare_fn = "equal"
										value_string = ["test"]
									}
									column {
										column_path  {
											name = "kind"
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
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.levels.0", "no_data"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.type", "webhook"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.action.0.description", "an interesting description 3 - reordered"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.1.send_reminders_interval", "33m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.action.0.description", "an interesting description 2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.send_reminders_interval", "22m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.conditions.0.operator", "or"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.conditions.0.compare_terms.0.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.conditions.0.compare_terms.0.column.0.column_path.0.name", "description"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.conditions.0.compare_terms.1.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.2.conditions.0.compare_terms.1.column.0.column_path.0.name", "kind"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "custom_variables", `{"fizz":"buzz"}`),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "max_alerts_per_hour", "0"),
				),
			},
		},
	})
}

func TestAccObserveMonitorIntervals(t *testing.T) {
	t.Skip("Skipping interval monitor tests - interval monitors are deprecated")
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

// monitorV2AlarmModeConfig renders a cron-scheduled threshold monitor whose
// scheduled block optionally carries an alarm_mode line. Pass "" to omit
// alarm_mode entirely (the SlowLane-safe path that must send no alarmMode).
func monitorV2AlarmModeConfig(prefix, alarmModeLine string) string {
	return fmt.Sprintf(monitorV2ConfigPreamble+`
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
					%[2]s
				}
			}
		}

		data "observe_monitor_v2" "lookup" {
			id = observe_monitor_v2.first.id
		}
	`, prefix, alarmModeLine)
}

// TestMonitorV2AlarmModeSchema locks in the schema decision (Abhinav's review on
// PR #334): alarm_mode is Optional with validation/diff-suppression, but must NOT
// carry a Default or be Computed. A Default would make the provider send an
// explicit alarmMode for every cron monitor, which the backend rejects on tenants
// without the FlagEnableOngoingAlarms feature flag.
func TestMonitorV2AlarmModeSchema(t *testing.T) {
	scheduled := resourceMonitorV2().Schema["scheduling"].Elem.(*schema.Resource).
		Schema["scheduled"].Elem.(*schema.Resource).Schema
	field, ok := scheduled["alarm_mode"]
	if !ok {
		t.Fatal("alarm_mode field missing from the scheduled schema")
	}
	if !field.Optional {
		t.Error("alarm_mode should be Optional")
	}
	if field.Computed {
		t.Error("alarm_mode must NOT be Computed: we deliberately send nothing when unset so flagless tenants are not rejected")
	}
	if field.Default != nil {
		t.Errorf("alarm_mode must NOT have a Default (got %v): a default forces an explicit alarmMode the backend rejects without the feature flag", field.Default)
	}
	if field.ValidateDiagFunc == nil {
		t.Error("alarm_mode should have a ValidateDiagFunc")
	}
	if field.DiffSuppressFunc == nil {
		t.Error("alarm_mode should have a DiffSuppressFunc")
	}
}

// TestNewMonitorV2ScheduledScheduleInput_AlarmMode verifies the expander only
// sends alarmMode when the user explicitly set it. The unset case returning nil
// is the crux of the PR #334 fix and is checked without a backend or feature flag.
func TestNewMonitorV2ScheduledScheduleInput_AlarmMode(t *testing.T) {
	amPtr := func(m gql.MonitorV2AlarmMode) *gql.MonitorV2AlarmMode { return &m }

	cases := []struct {
		name      string
		alarmMode interface{} // nil => omit from config
		want      *gql.MonitorV2AlarmMode
	}{
		{"unset", nil, nil},
		{"ongoing", "ongoing", amPtr(gql.MonitorV2AlarmModeOngoing)},
		{"per_run", "per_run", amPtr(gql.MonitorV2AlarmModePerrun)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheduled := map[string]interface{}{
				"timezone": "America/New_York",
				"raw_cron": "0 0 * * *",
			}
			if tc.alarmMode != nil {
				scheduled["alarm_mode"] = tc.alarmMode
			}
			data := schema.TestResourceDataRaw(t, resourceMonitorV2().Schema, map[string]interface{}{
				"scheduling": []interface{}{
					map[string]interface{}{
						"scheduled": []interface{}{scheduled},
					},
				},
			})

			cron, diags := newMonitorV2ScheduledScheduleInput("scheduling.0.scheduled.0.", data)
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			switch {
			case tc.want == nil:
				if cron.AlarmMode != nil {
					t.Fatalf("expected AlarmMode to be nil when unset, got %q", *cron.AlarmMode)
				}
			case cron.AlarmMode == nil:
				t.Fatalf("expected AlarmMode %q, got nil", *tc.want)
			case *cron.AlarmMode != *tc.want:
				t.Fatalf("expected AlarmMode %q, got %q", *tc.want, *cron.AlarmMode)
			}
		})
	}
}

// TestMonitorV2FlattenScheduledSchedule_AlarmMode verifies the read path: a nil
// backend value leaves alarm_mode absent (so config-unset round-trips with no
// diff), and a set value is snake-cased back into state.
func TestMonitorV2FlattenScheduledSchedule_AlarmMode(t *testing.T) {
	amPtr := func(m gql.MonitorV2AlarmMode) *gql.MonitorV2AlarmMode { return &m }

	got := monitorV2FlattenScheduledSchedule(gql.MonitorV2CronSchedule{Timezone: "UTC"})
	if _, ok := got[0].(map[string]any)["alarm_mode"]; ok {
		t.Errorf("expected no alarm_mode key when AlarmMode is nil")
	}

	for _, tc := range []struct {
		mode gql.MonitorV2AlarmMode
		want string
	}{
		{gql.MonitorV2AlarmModeOngoing, "ongoing"},
		{gql.MonitorV2AlarmModePerrun, "per_run"},
	} {
		got := monitorV2FlattenScheduledSchedule(gql.MonitorV2CronSchedule{
			Timezone:  "UTC",
			AlarmMode: amPtr(tc.mode),
		})
		if v := got[0].(map[string]any)["alarm_mode"]; v != tc.want {
			t.Errorf("AlarmMode %q: expected flattened %q, got %v", tc.mode, tc.want, v)
		}
	}
}

// TestAccObserveMonitorV2AlarmMode exercises the alarm_mode lifecycle end to end:
// the unset default path, setting/changing the value, case-insensitive diff
// suppression, and clearing the value back to nil.
//
// NOTE: the steps that APPLY a non-nil alarm_mode (ongoing/per_run) require the
// FlagEnableOngoingAlarms feature flag on the test tenant; otherwise the backend
// returns "alarmMode is not enabled for this customer". The omit/plan-only steps
// do not need the flag.
func TestAccObserveMonitorV2AlarmMode(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	noAlarm := monitorV2AlarmModeConfig(randomPrefix, "")
	ongoing := monitorV2AlarmModeConfig(randomPrefix, `alarm_mode = "ongoing"`)
	perRun := monitorV2AlarmModeConfig(randomPrefix, `alarm_mode = "per_run"`)
	ongoingPascal := monitorV2AlarmModeConfig(randomPrefix, `alarm_mode = "Ongoing"`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Default path: no alarm_mode set -> provider sends no alarmMode,
				// backend returns nil, state is empty. Safe on flagless tenants.
				Config: noAlarm,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.alarm_mode", ""),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.alarm_mode", ""),
				),
			},
			// Re-applying the same config produces no diff.
			testAccPlanOnlyNoDriftStep(noAlarm),
			{
				// Explicitly set ongoing (requires feature flag).
				Config: ongoing,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.alarm_mode", "ongoing"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.alarm_mode", "ongoing"),
				),
			},
			// Case-insensitive: "Ongoing" must diff-suppress against stored "ongoing".
			testAccPlanOnlyNoDriftStep(ongoingPascal),
			{
				// Change the value (requires feature flag).
				Config: perRun,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.alarm_mode", "per_run"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.alarm_mode", "per_run"),
				),
			},
			{
				// Removing alarm_mode clears it (the input fully replaces the
				// stored value), leaving state empty with no perpetual diff.
				Config: noAlarm,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "scheduling.0.scheduled.0.alarm_mode", ""),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.scheduled.0.alarm_mode", ""),
				),
			},
		},
	})
}

// TestAccObserveMonitorV2AlarmModeInvalid verifies an unknown alarm_mode is
// rejected at plan time by validateEnums (no backend write, no feature flag).
func TestAccObserveMonitorV2AlarmModeInvalid(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      monitorV2AlarmModeConfig(randomPrefix, `alarm_mode = "bogus"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("to be one of"),
			},
		},
	})
}

// monitorV2ServiceBindingsConfig renders a transform-scheduled threshold monitor
// that optionally carries a service_bindings block (pass "" to omit it).
func monitorV2ServiceBindingsConfig(prefix, bindingBlock string) string {
	return fmt.Sprintf(monitorV2ConfigPreamble+`
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
			%[2]s
			scheduling {
				transform {
					freshness_goal = "15m"
				}
			}
		}

		data "observe_monitor_v2" "lookup" {
			id = observe_monitor_v2.first.id
		}
	`, prefix, bindingBlock)
}

// monitorV2ServiceBindingsWildcardConfig renders a monitor with an all-wildcard
// service binding. A wildcard dimension resolves its value at detection time from a
// correlation-tag grouping, so the input dataset must carry the canonical correlation
// tags (service.name / deployment.environment.name / service.namespace) on real
// columns and the monitor must group by those tags. This requires, on the tenant:
// the FlagEnableServiceMonitoring feature flag AND the customer-scoped
// Compiler.propagateCorrelationTags layered setting (so the tags propagate from the
// input dataset into the monitor's compiled schema).
func monitorV2ServiceBindingsWildcardConfig(prefix string, nsWildcard bool) string {
	// When nsWildcard is false the service_namespace dimension is exact, so its
	// backing correlation tag, grouping, and depends_on entry are all dropped.
	nsTagResource := `

		resource "observe_correlation_tag" "ns" {
			dataset = observe_dataset.tagged.oid
			column  = "ns_name"
			name    = "service.namespace"
		}`
	nsDependsOn := `
				observe_correlation_tag.ns,`
	nsGrouping := `
			groupings {
				correlation_tag {
					tag = "service.namespace"
				}
			}`
	nsBinding := `
				service_namespace {
					match_mode = "wildcard"
				}`
	if !nsWildcard {
		nsTagResource = ""
		nsDependsOn = ""
		nsGrouping = ""
		nsBinding = `
				service_namespace {
					value = "payments"
				}`
	}

	return fmt.Sprintf(monitorV2ConfigPreamble+`
		resource "observe_dataset" "tagged" {
			workspace = data.observe_workspace.default.oid
			name      = "%[1]s-tagged"
			inputs    = { "test" = observe_datastream.test.dataset }
			stage {
				pipeline = <<-EOF
					filter false
					colmake svc_name:string("svc"), env_name:string("env"), ns_name:string("ns")
				EOF
			}
		}

		resource "observe_correlation_tag" "svc" {
			dataset = observe_dataset.tagged.oid
			column  = "svc_name"
			name    = "service.name"
		}
		resource "observe_correlation_tag" "env" {
			dataset = observe_dataset.tagged.oid
			column  = "env_name"
			name    = "deployment.environment.name"
		}%[2]s

		resource "observe_monitor_v2" "first" {
			workspace     = data.observe_workspace.default.oid
			rule_kind     = "threshold"
			name          = "%[1]s"
			lookback_time = "10m"
			inputs        = { "test" = observe_dataset.tagged.oid }
			depends_on = [
				observe_correlation_tag.svc,
				observe_correlation_tag.env,%[3]s
			]
			stage {
				pipeline = "colmake temp_number:14"
			}
			rules {
				level = "informational"
				threshold {
					compare_values {
						compare_fn  = "greater"
						value_int64 = [0]
					}
					value_column_name = "temp_number"
					aggregation       = "all_of"
				}
			}
			groupings {
				correlation_tag {
					tag = "service.name"
				}
			}
			groupings {
				correlation_tag {
					tag = "deployment.environment.name"
				}
			}%[4]s
			service_bindings {
				service_name {
					match_mode = "wildcard"
				}
				environment {
					match_mode = "wildcard"
				}%[5]s
			}
			scheduling {
				transform {
					freshness_goal = "15m"
				}
			}
		}

		data "observe_monitor_v2" "lookup" {
			id = observe_monitor_v2.first.id
		}
	`, prefix, nsTagResource, nsDependsOn, nsGrouping, nsBinding)
}

// TestMonitorV2ServiceBindingSchema locks in the schema shape: service_bindings is
// an Optional block capped at one entry, each of the three dimensions is a Required
// single-entry block, and match_mode defaults to "exact" with enum validation and
// case-insensitive diff suppression while value stays Optional.
func TestMonitorV2ServiceBindingSchema(t *testing.T) {
	sb, ok := resourceMonitorV2().Schema["service_bindings"]
	if !ok {
		t.Fatal("service_bindings field missing from the monitor v2 schema")
	}
	if !sb.Optional {
		t.Error("service_bindings should be Optional")
	}
	if sb.MaxItems != 1 {
		t.Errorf("service_bindings should have MaxItems=1, got %d", sb.MaxItems)
	}

	dimensions := sb.Elem.(*schema.Resource).Schema
	for _, name := range []string{"service_name", "environment", "service_namespace"} {
		dim, ok := dimensions[name]
		if !ok {
			t.Fatalf("service_bindings.%s missing", name)
		}
		if !dim.Required {
			t.Errorf("service_bindings.%s should be Required", name)
		}
		if dim.MinItems != 1 || dim.MaxItems != 1 {
			t.Errorf("service_bindings.%s should have MinItems=1 MaxItems=1, got %d/%d", name, dim.MinItems, dim.MaxItems)
		}

		value := dim.Elem.(*schema.Resource).Schema
		matchMode := value["match_mode"]
		if matchMode == nil {
			t.Errorf("service_bindings.%s.match_mode missing", name)
			continue
		}
		if !matchMode.Optional {
			t.Errorf("service_bindings.%s.match_mode should be Optional", name)
		}
		if matchMode.Default != "exact" {
			t.Errorf("service_bindings.%s.match_mode should default to \"exact\", got %v", name, matchMode.Default)
		}
		if matchMode.ValidateDiagFunc == nil {
			t.Errorf("service_bindings.%s.match_mode should have a ValidateDiagFunc", name)
		}
		if matchMode.DiffSuppressFunc == nil {
			t.Errorf("service_bindings.%s.match_mode should have a DiffSuppressFunc", name)
		}
		if v := value["value"]; v == nil || !v.Optional {
			t.Errorf("service_bindings.%s.value should be Optional", name)
		}
	}
}

// TestMonitorV2CorrelationTagSchema verifies the correlation_tag column variant is
// an Optional single-entry block with a Required tag, sitting alongside link_column
// and column_path.
func TestMonitorV2CorrelationTagSchema(t *testing.T) {
	column := monitorV2ColumnResource().Schema
	ct, ok := column["correlation_tag"]
	if !ok {
		t.Fatal("correlation_tag field missing from the column schema")
	}
	if !ct.Optional {
		t.Error("correlation_tag should be Optional")
	}
	if ct.MaxItems != 1 {
		t.Errorf("correlation_tag should have MaxItems=1, got %d", ct.MaxItems)
	}
	if tag := ct.Elem.(*schema.Resource).Schema["tag"]; tag == nil || !tag.Required {
		t.Error("correlation_tag.tag should be Required")
	}
}

// TestNewMonitorV2ServiceBindingInput verifies the expander maps each dimension's
// value/match_mode into the GraphQL input, including a mixed exact/wildcard binding.
func TestNewMonitorV2ServiceBindingInput(t *testing.T) {
	sp := func(s string) *string { return &s }
	exact := gql.MonitorV2ServiceBindingMatchModeExact
	wildcard := gql.MonitorV2ServiceBindingMatchModeWildcard

	cases := []struct {
		name    string
		binding map[string]interface{}
		want    gql.MonitorV2ServiceBindingInput
	}{
		{
			name: "all exact",
			binding: map[string]interface{}{
				"service_name":      []interface{}{map[string]interface{}{"value": "checkout", "match_mode": "exact"}},
				"environment":       []interface{}{map[string]interface{}{"value": "prod", "match_mode": "exact"}},
				"service_namespace": []interface{}{map[string]interface{}{"value": "payments", "match_mode": "exact"}},
			},
			want: gql.MonitorV2ServiceBindingInput{
				ServiceName:      gql.MonitorV2ServiceBindingValueInput{Value: sp("checkout"), MatchMode: &exact},
				Environment:      gql.MonitorV2ServiceBindingValueInput{Value: sp("prod"), MatchMode: &exact},
				ServiceNamespace: gql.MonitorV2ServiceBindingValueInput{Value: sp("payments"), MatchMode: &exact},
			},
		},
		{
			name: "mixed wildcard",
			binding: map[string]interface{}{
				"service_name":      []interface{}{map[string]interface{}{"match_mode": "wildcard"}},
				"environment":       []interface{}{map[string]interface{}{"match_mode": "wildcard"}},
				"service_namespace": []interface{}{map[string]interface{}{"value": "payments", "match_mode": "exact"}},
			},
			want: gql.MonitorV2ServiceBindingInput{
				ServiceName:      gql.MonitorV2ServiceBindingValueInput{MatchMode: &wildcard},
				Environment:      gql.MonitorV2ServiceBindingValueInput{MatchMode: &wildcard},
				ServiceNamespace: gql.MonitorV2ServiceBindingValueInput{Value: sp("payments"), MatchMode: &exact},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceMonitorV2().Schema, map[string]interface{}{
				"service_bindings": []interface{}{tc.binding},
			})
			got, diags := newMonitorV2ServiceBindingInput("service_bindings.0.", data)
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("expected %+v, got %+v", tc.want, *got)
			}
		})
	}
}

// TestNewMonitorV2ServiceBindingValueInput_Validation mirrors the backend's
// value/match_mode pairing rule client-side: exact requires a value, wildcard
// forbids one. Checked without a backend or feature flag.
func TestNewMonitorV2ServiceBindingValueInput_Validation(t *testing.T) {
	cases := []struct {
		name    string
		dim     map[string]interface{}
		wantErr bool
	}{
		{"exact with value", map[string]interface{}{"match_mode": "exact", "value": "x"}, false},
		{"exact without value", map[string]interface{}{"match_mode": "exact"}, true},
		{"wildcard without value", map[string]interface{}{"match_mode": "wildcard"}, false},
		{"wildcard with value", map[string]interface{}{"match_mode": "wildcard", "value": "x"}, true},
		// match_mode omitted -> schema Default "exact" applies, so a value is required and accepted.
		{"omitted defaults to exact", map[string]interface{}{"value": "x"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceMonitorV2().Schema, map[string]interface{}{
				"service_bindings": []interface{}{
					map[string]interface{}{
						"service_name": []interface{}{tc.dim},
					},
				},
			})
			_, diags := newMonitorV2ServiceBindingValueInput("service_bindings.0.service_name.0.", data)
			if tc.wantErr && !diags.HasError() {
				t.Errorf("expected a validation error, got none")
			}
			if !tc.wantErr && diags.HasError() {
				t.Errorf("unexpected validation error: %v", diags)
			}
		})
	}
}

// TestMonitorV2FlattenServiceBindings verifies the read path snake-cases match_mode
// and omits value for wildcard dimensions.
func TestMonitorV2FlattenServiceBindings(t *testing.T) {
	sp := func(s string) *string { return &s }
	exact := gql.MonitorV2ServiceBindingMatchModeExact
	wildcard := gql.MonitorV2ServiceBindingMatchModeWildcard

	got := monitorV2FlattenServiceBindings([]gql.MonitorV2ServiceBinding{
		{
			ServiceName:      gql.MonitorV2ServiceBindingValue{Value: sp("checkout"), MatchMode: &exact},
			Environment:      gql.MonitorV2ServiceBindingValue{MatchMode: &wildcard},
			ServiceNamespace: gql.MonitorV2ServiceBindingValue{Value: sp("payments"), MatchMode: &exact},
		},
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(got))
	}
	binding := got[0].(map[string]interface{})

	serviceName := binding["service_name"].([]interface{})[0].(map[string]interface{})
	if serviceName["value"] != "checkout" || serviceName["match_mode"] != "exact" {
		t.Errorf("service_name flattened wrong: %+v", serviceName)
	}

	environment := binding["environment"].([]interface{})[0].(map[string]interface{})
	if _, ok := environment["value"]; ok {
		t.Errorf("environment should have no value for wildcard, got %+v", environment)
	}
	if environment["match_mode"] != "wildcard" {
		t.Errorf("environment match_mode should be wildcard, got %v", environment["match_mode"])
	}
}

// TestMonitorV2CorrelationTag_ExpandFlatten round-trips the correlation_tag column
// variant through the expander and flattener.
func TestMonitorV2CorrelationTag_ExpandFlatten(t *testing.T) {
	// Expand: a column whose only variant is a correlation_tag grouping.
	data := schema.TestResourceDataRaw(t, resourceMonitorV2().Schema, map[string]interface{}{
		"groupings": []interface{}{
			map[string]interface{}{
				"correlation_tag": []interface{}{map[string]interface{}{"tag": "service.name"}},
			},
		},
	})
	col, diags := newMonitorV2ColumnInput("groupings.0.", data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if col.CorrelationTag == nil || col.CorrelationTag.Tag != "service.name" {
		t.Fatalf("expected correlation_tag input with tag=service.name, got %+v", col.CorrelationTag)
	}
	if col.LinkColumn != nil || col.ColumnPath != nil {
		t.Errorf("expected only correlation_tag to be set, got link=%v path=%v", col.LinkColumn, col.ColumnPath)
	}

	// Flatten: the read struct back into terraform state shape.
	got := monitorV2FlattenCorrelationTag(gql.MonitorV2CorrelationTag{Tag: "service.name"})
	want := []interface{}{map[string]interface{}{"tag": "service.name"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

// TestAccObserveMonitorV2ServiceBindingsExact exercises the exact-match service
// binding lifecycle: create, read-back via the data source, no-drift re-plan, and
// clearing the binding by removing the block.
//
// NOTE: requires the FlagEnableServiceMonitoring feature flag on the test tenant;
// otherwise the backend rejects any service binding.
func TestAccObserveMonitorV2ServiceBindingsExact(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	bindings := `
		service_bindings {
			service_name {
				value = "checkout"
			}
			environment {
				value      = "prod"
				match_mode = "exact"
			}
			service_namespace {
				value = "payments"
			}
		}`
	withBinding := monitorV2ServiceBindingsConfig(randomPrefix, bindings)
	updatedBindings := `
		service_bindings {
			service_name {
				value = "frontend"
			}
			environment {
				value      = "staging"
				match_mode = "exact"
			}
			service_namespace {
				value = "checkout"
			}
		}`
	withUpdatedBinding := monitorV2ServiceBindingsConfig(randomPrefix, updatedBindings)
	noBinding := monitorV2ServiceBindingsConfig(randomPrefix, "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withBinding,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.value", "checkout"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.match_mode", "exact"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.environment.0.value", "prod"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_namespace.0.value", "payments"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.service_name.0.value", "checkout"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.service_namespace.0.value", "payments"),
				),
			},
			// Re-applying the same config produces no diff.
			testAccPlanOnlyNoDriftStep(withBinding),
			{
				// Update: changing the binding's values updates it in place.
				Config: withUpdatedBinding,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.value", "frontend"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.environment.0.value", "staging"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_namespace.0.value", "checkout"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.service_name.0.value", "frontend"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.environment.0.value", "staging"),
				),
			},
			testAccPlanOnlyNoDriftStep(withUpdatedBinding),
			{
				// Removing the block clears the binding (input fully replaces stored bindings).
				Config: noBinding,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.#", "0"),
				),
			},
		},
	})
}

// TestAccObserveMonitorV2ServiceBindingsWildcard exercises an all-wildcard binding
// backed by correlation_tag groupings over a dataset whose columns are correlation-
// tagged with the canonical tags.
//
// NOTE: requires, on the tenant, the FlagEnableServiceMonitoring feature flag AND the
// customer-scoped Compiler.propagateCorrelationTags layered setting enabled, so the
// input dataset's correlation tags propagate into the monitor's compiled schema and
// the wildcard dimensions validate at save time.
func TestAccObserveMonitorV2ServiceBindingsWildcard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	allWildcard := monitorV2ServiceBindingsWildcardConfig(randomPrefix, true)
	nsExact := monitorV2ServiceBindingsWildcardConfig(randomPrefix, false)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: allWildcard,
				// Creating a monitor for a new dataset updates the dataset's
				// freshness configuration on first save, which bumps the version
				// timestamp in its OID. The observe_correlation_tag resources
				// store the full versioned OID, so they show a ForceNew diff on
				// the next plan. Subsequent monitor updates are idempotent.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.match_mode", "wildcard"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.value", ""),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.environment.0.match_mode", "wildcard"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_namespace.0.match_mode", "wildcard"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "groupings.#", "3"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "groupings.0.correlation_tag.0.tag", "service.name"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.service_name.0.match_mode", "wildcard"),
				),
			},
			{
				// Update: flip service_namespace wildcard->exact, which lets us drop its
				// correlation_tag grouping and backing tag. Exercises both a binding update
				// and a change to the set of correlation-tag groupings (3 -> 2).
				Config: nsExact,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_name.0.match_mode", "wildcard"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.environment.0.match_mode", "wildcard"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_namespace.0.match_mode", "exact"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "service_bindings.0.service_namespace.0.value", "payments"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "groupings.#", "2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "groupings.1.correlation_tag.0.tag", "deployment.environment.name"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "service_bindings.0.service_namespace.0.value", "payments"),
				),
			},
		},
	})
}

// TestAccObserveMonitorV2ServiceBindingsInvalid verifies the client-side
// value/match_mode pairing rule fails the apply before any backend call (so no
// feature flag is needed): a wildcard dimension may not carry a value.
func TestAccObserveMonitorV2ServiceBindingsInvalid(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	bindings := `
		service_bindings {
			service_name {
				match_mode = "wildcard"
				value      = "checkout"
			}
			environment {
				value = "prod"
			}
			service_namespace {
				value = "payments"
			}
		}`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      monitorV2ServiceBindingsConfig(randomPrefix, bindings),
				ExpectError: regexp.MustCompile("match_mode=wildcard must not set value"),
			},
		},
	})
}

// Tests that converting an inline action to a shared action does not cause an error.
// We've had issues with this in the past due to how the Terraform SDK handles
// testing for existence of object types that previously had a value and now don't.
// See newMonitorV2ActionAndRelation() for more details.
func TestAccObserveMonitorInlineToSharedAction(t *testing.T) {
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
						lookback_time = "10m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "filter true"
						}
						rules {
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
									value_int64 = [100000000]
								}
							}
						}
						actions {
							action {
								type = "email"
								email {
									subject = "inline action"
									addresses = ["test@observeinc.com"]
								}
								description = "an interesting description 1"
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.type", "email"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "actions.0.action.0.description", "an interesting description 1"),
				),
			},
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "shared action"
							addresses = ["test@observeinc.com"]
						}
						name = "%[1]s"
					}

					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[1]s"
						lookback_time = "10m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "filter true"
						}
						rules {
							level = "informational"
							count {
								compare_values {
									compare_fn = "greater"
									value_int64 = [100000000]
								}
							}
						}
						actions {
							oid = observe_monitor_v2_action.act.oid
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "actions.0.oid"),
					resource.TestCheckNoResourceAttr("observe_monitor_v2.first", "actions.0.action.0.type"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2CompareAgainstZeroVals(t *testing.T) {
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
							pipeline = "colmake temp_number:14"
						}
						rules {
							level = "informational"
							promote {
								compare_columns {
									compare_values {
										compare_fn = "equal"
										value_int64 = [0]
									}
									compare_values {
										compare_fn = "equal"
										value_float64 = [0.0]
									}
									compare_values {
										compare_fn = "equal"
										value_bool = [false]
									}
									compare_values {
										compare_fn = "equal"
										value_string = [""]
									}
									compare_values {
										compare_fn = "equal"
										value_duration = ["0s"]
									}
									compare_values {
										compare_fn = "equal"
										value_timestamp = ["1970-01-01T00:00:00Z"]
									}
									column {
										column_path {
											name = "temp_number"
										}
									}
								}
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.1.value_float64.0", "0"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.2.value_bool.0", "false"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.3.value_string.0", ""),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.4.value_duration.0", "0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.promote.0.compare_columns.0.compare_values.5.value_timestamp.0", "1970-01-01T00:00:00Z"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2Anomaly(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "anomaly"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14"
						}
						stage {
							pipeline = "timechart 5m, temp_number:avg(temp_number)"
						}
						rule_template {
							anomaly {
								value_column_name = "temp_number"
								compare_fn = "above"
								num_standard_deviations = 3
								basic_algorithm {}
							}
						}
						rules {
							level = "informational"
							anomaly {
								compare_percentage = 50
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "anomaly"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.compare_fn", "above"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.num_standard_deviations", "3"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.basic_algorithm.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.anomaly.0.compare_percentage", "50"),
				),
			},
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "anomaly"
						name = "%[1]s"
						lookback_time = "1h"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14"
						}
						stage {
							pipeline = "timechart 5m, temp_number:avg(temp_number)"
						}
						rule_template {
							anomaly {
								value_column_name = "temp_number"
								compare_fn = "above_or_below"
								num_standard_deviations = 2
								basic_algorithm {}
							}
						}
						rules {
							level = "informational"
							anomaly {
								compare_percentage = 25
							}
						}
						rules {
							level = "warning"
							anomaly {
								compare_percentage = 75
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "lookback_time", "1h0m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "anomaly"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.compare_fn", "above_or_below"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.num_standard_deviations", "2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.anomaly.0.compare_percentage", "25"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.level", "warning"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.1.anomaly.0.compare_percentage", "75"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2AnomalyWithNoDataRule(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "anomaly"
						name = "%[1]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14"
						}
						stage {
							pipeline = "timechart 5m, temp_number:avg(temp_number)"
						}
						rule_template {
							anomaly {
								value_column_name = "temp_number"
								compare_fn = "below"
								num_standard_deviations = 2
								basic_algorithm {}
							}
						}
						no_data_rules {
							expiration = "30m"
							anomaly {}
						}
						rules {
							level = "informational"
							anomaly {
								compare_percentage = 50
							}
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor_v2.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_kind", "anomaly"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.compare_fn", "below"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rule_template.0.anomaly.0.num_standard_deviations", "2"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "no_data_rules.0.expiration", "30m0s"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("observe_monitor_v2.first", "rules.0.anomaly.0.compare_percentage", "50"),
				),
			},
		},
	})
}
