package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveGetIDMonitorV2CountData(t *testing.T) {
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

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.oid
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "count"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.count.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.count.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.interval", "15m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.randomize", "0s"),
				),
			},
		},
	})
}

func TestAccObserveGetIDMonitorV2Threshold(t *testing.T) {
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
						comment = "a descriptive comment"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14"
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
								interval = "15m"
								randomize = "0"
							}
						}
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.oid
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "threshold"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.aggregation", "all_of"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.interval", "15m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.randomize", "0s"),
				),
			},
		},
	})
}

func TestAccObserveGetIDMonitorV2Promote(t *testing.T) {
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
						comment = "a descriptive comment"
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
						scheduling {
							interval {
								interval = "15m"
								randomize = "0"
							}
						}
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.oid
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "promote"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.compare_values.0.value_int64.0", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.column.0.column_path.0.name", "temp_number"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.interval", "15m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.interval.0.randomize", "0s"),
				),
			},
		},
	})
}
