package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var monitorConfigPreamble = configPreamble + datastreamConfigPreamble

func TestAccObserveMonitor(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					freshness = "4m"

					comment = "a descriptive comment"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					definition = jsonencode({ "hello" = "world" })

					stage {}

					rule {
						count {
							compare_function   = "less_or_equal"
							compare_values     = [1]
							lookback_time      = "1m"
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "freshness", "4m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "comment", "a descriptive comment"),
					resource.TestCheckResourceAttr("observe_monitor.first", "definition", `{"hello":"world"}`),
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.compare_function", "less_or_equal"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.reminder_frequency", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_reminder", "false"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_close", "false"),
				),
			},
			//			{
			//				Config: fmt.Sprintf(monitorConfigPreamble +`
			//				resource "observe_monitor" "first" {
			//					workspace = data.observe_workspace.default.oid
			//					name      = "%[1]s"
			//
			//					inputs = {
			//						"test" = observe_datastream.test.dataset
			//					}
			//
			//					stage {
			//						pipeline = "filter false"
			//					}
			//
			//					rule {
			//						source_column = "OBSERVATION_KIND"
			//
			//						change {
			//							aggregate_function = "sum"
			//							compare_function   = "greater"
			//							compare_value      = 100
			//							lookback_time      = "1m"
			//							baseline_time      = "2m"
			//						}
			//					}
			//
			//					notification_spec {
			//						importance = "important"
			//						merge      = "separate"
			//					}
			//				}`, randomPrefix),
			//				// compare_value is deprecated, so compare_values will also be populated
			//				ExpectNonEmptyPlan: true,
			//				Check: resource.ComposeTestCheckFunc(
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "stage.0.pipeline"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.count.0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_function", "greater"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_values.0", "100"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.lookback_time", "1m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.baseline_time", "2m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "important"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "separate"),
			//				),
			//			},
			//			{
			//				Config: fmt.Sprintf(monitorConfigPreamble +`
			//				resource "observe_monitor" "first" {
			//					workspace = data.observe_workspace.default.oid
			//					name      = "%[1]s"
			//
			//					inputs = {
			//						"test" = observe_datastream.test.dataset
			//					}
			//
			//					stage {
			//						pipeline = "filter false"
			//					}
			//
			//					rule {
			//						source_column = "OBSERVATION_KIND"
			//
			//						change {
			//							aggregate_function = "sum"
			//							compare_function   = "greater"
			//							compare_values     = [ 0 ]
			//							lookback_time      = "1m"
			//							baseline_time      = "2m"
			//						}
			//					}
			//
			//					notification_spec {
			//						importance = "important"
			//						merge      = "separate"
			//					}
			//				}`, randomPrefix),
			//				Check: resource.ComposeTestCheckFunc(
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "stage.0.pipeline"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.count.0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_function", "greater"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_values.0", "0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.lookback_time", "1m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.baseline_time", "2m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "important"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "separate"),
			//				),
			//			},
		},
	})
}

func TestAccObserveMonitorThreshold(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "colmake temp_number:14"
					}


					rule {
                        source_column    = "temp_number"

						threshold {
                            compare_function = "greater"
                            compare_values   = [ 70 ]
                            lookback_time    = "10m0s"
						}
					}

					notification_spec {
						merge = "merged"
						reminder_frequency = "5m"
						notify_on_close = true
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", "colmake temp_number:14"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_values.0", "70"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.lookback_time", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.threshold_agg_function", "at_all_times"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.reminder_frequency", "5m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_reminder", "true"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_close", "true"),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "false"),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "colmake temp_number:14"
					}


					rule {
                        source_column    = "temp_number"

						threshold {
                            compare_function       = "greater"
                            compare_values         = [ 70 ]
                            lookback_time          = "10m0s"
							threshold_agg_function = "at_least_once"
						}
					}

					notification_spec {
						merge = "merged"
						reminder_frequency = "5m"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", "colmake temp_number:14"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_values.0", "70"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.lookback_time", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.threshold_agg_function", "at_least_once"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "false"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_reminder", "true"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.reminder_frequency", "5m0s"),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "colmake temp_number:14"
					}


					rule {
                        source_column    = "temp_number"

						threshold {
                            compare_function = "greater"
                            compare_values   = [ 70 ]
                            lookback_time    = "10m0s"
						}
					}

					notification_spec {
						merge = "merged"
						notify_on_close = true
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.notify_on_reminder", "false"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.reminder_frequency", ""),
				),
			},
		},
	})
}

func TestAccObserveMonitorThresholdFloat(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "colmake temp_number:14"
					}


					rule {
                        source_column    = "temp_number"

						threshold {
                            compare_function = "greater"
                            compare_values   = [ 0.5 ]
                            lookback_time    = "10m0s"
						}
					}

					notification_spec {
						merge = "merged"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.test"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", "colmake temp_number:14"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_values.0", "0.5"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.lookback_time", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.threshold_agg_function", "at_all_times"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "false"),
				),
			},
		},
	})
}

func TestAccObserveMonitorFacetUpdate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							make_col test:string(FIELDS.text)
							make_resource OBSERVATION_KIND, primary_key(test)
						EOF
					}

					rule {
						source_column = "OBSERVATION_KIND"

						facet {
							facet_function = "equals"
							facet_values   = ["OBSERVATION_KIND"]
							time_function  = "never"
							lookback_time  = "1m"
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_function", "equals"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_values.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_function", "never"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.lookback_time", "1m0s"),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "filter false"
					}

					rule {
						source_column = "OBSERVATION_KIND"

						facet {
							facet_function = "equals"
							facet_values   = ["OBSERVATION_KIND"]
							time_function  = "at_least_once"
							time_value     = 0.555
							lookback_time  = "1m"
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_function", "equals"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_values.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_function", "at_least_once"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_value", "0.555"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.lookback_time", "1m0s"),
				),
			},
		},
	})
}

func TestAccObserveMonitorFacetCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = "filter false"
					}

					rule {
						source_column = "OBSERVATION_KIND"

						facet {
							facet_function = "is_null"
							facet_values   = []
							time_function  = "at_least_once"
							time_value     = 0.555
							lookback_time  = "1m"
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_function", "is_null"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_values.#", "0"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_function", "at_least_once"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_value", "0.555"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.lookback_time", "1m0s"),
				),
			},
		},
	})
}

func TestAccObserveMonitorPromote(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							colmake kind:"test", description:"test"
						EOF
					}

					rule {
						group_by_group {}

						promote {
							primary_key       = ["OBSERVATION_KIND"]
							kind_field        = "kind"
							description_field = "description"
						}

					}

					notification_spec {
						merge      = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.columns.#", "0"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", "kind"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", "description"),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					disabled  = true

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					rule {
						group_by_group {}

						promote {
							primary_key       = ["OBSERVATION_KIND"]
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.columns.#", "0"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
		},
	})
}
func TestAccObserveMonitorLog(t *testing.T) {
	t.Skip()
	t.Skip()
	// TODO(OB-26540) Some optional monitor fields can't be updated to null

	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
						make_col vt:BUNDLE_TIMESTAMP
						make_interval vt
						EOF
					}
				}

				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_dataset.first.oid
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

					rule {
						source_column = "OBSERVATION_INDEX"

						log {
							compare_function   = "greater"
							compare_values     = [1]
							lookback_time      = "1m"
						}
					}

					notification_spec {
						merge      = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.expression_summary", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.log_stage_id", ""),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
						make_col vt:BUNDLE_TIMESTAMP
						make_interval vt
						EOF
					}
				}

				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_dataset.first.oid
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

					rule {
						source_column = "OBSERVATION_INDEX"

						log {
							compare_function   = "greater"
							compare_values     = [1]
							lookback_time      = "1m"
							expression_summary = "Some text"
							log_stage_id = "stage-1"
						}
					}

					notification_spec {
						merge      = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.expression_summary", "Some text"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.log_stage_id", "stage-1"),
				),
			},
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_dataset" "first" {
					workspace                        = data.observe_workspace.default.oid
					name 	                         = "%[1]s-first"

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
						make_col vt:BUNDLE_TIMESTAMP
						make_interval vt
						EOF
					}
				}

				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"

					inputs = {
						"test" = observe_dataset.first.oid
					}

					stage {
						pipeline = <<-EOF
							filter OBSERVATION_INDEX != 0
						EOF
					}
					stage {
						pipeline = "timechart 1m, frame(back:10m), A_ContainerLogsClean_count:count(), group_by()"
					}

					rule {
						source_column = "A_ContainerLogsClean_count"

						log {
							compare_function   = "greater"
							compare_values     = [1]
							lookback_time      = "1m"
							expression_summary = "Some text"
							source_log_dataset = observe_dataset.first.oid
							log_stage_id = "stage-0"
						}
					}

					notification_spec {
						merge      = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.expression_summary", "Some text"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.log.0.log_stage_id", "stage-0"),
					resource.TestCheckResourceAttrPair("observe_monitor.first", "rule.0.log.0.source_log_dataset", "observe_dataset.first", "oid"),
				),
			},
		},
	})
}

func TestAccObserveMonitorGroupByGroup(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					disabled  = true

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					rule {
						group_by_group {
							columns = ["OBSERVATION_KIND"]
						}

						promote {
							primary_key       = ["OBSERVATION_KIND"]
						}
					}

					notification_spec {
						merge       = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.columns.0", "OBSERVATION_KIND"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
		},
	})
}

func TestAccObserveMonitorGroupByGroupEmpty(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					disabled  = true

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
							timechart 1h, Count:count(1), group_by(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}

					rule {
						group_by_group {
						}

						promote {
							primary_key       = []
							kind_field			= "OBSERVATION_KIND"
						}
					}

					notification_spec {
						merge       = "separate"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.columns.#", "0"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "0"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", "OBSERVATION_KIND"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
			{
				// Empty columns var produces no change
				PlanOnly: true,
				Config: fmt.Sprintf(monitorConfigPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					disabled  = true

					inputs = {
						"test" = observe_datastream.test.dataset
					}

					stage {
						pipeline = <<-EOF
							filter true
							timechart 1h, Count:count(1), group_by(OBSERVATION_KIND, BUNDLE_ID)
						EOF
					}

					rule {
						group_by_group {
							columns = []
						}

						promote {
							primary_key       = []
							kind_field			= "OBSERVATION_KIND"
						}
					}

					notification_spec {
						merge       = "separate"
					}
				}`, randomPrefix),
			},
		},
	})
}
