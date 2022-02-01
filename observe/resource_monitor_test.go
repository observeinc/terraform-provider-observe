package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveMonitor(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {}

					rule {
						group_by      = "none"

						count {
							compare_function   = "less_or_equal"
							compare_values     = [1]
							lookback_time      = "1m"
						}
					}

					notification_spec {
						selection       = "count"
						selection_value = 1
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_monitor.first", "workspace"),
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.compare_function", "less_or_equal"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.count.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.selection", "count"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.selection_value", "1"),
				),
			},
			//			{
			//				Config: fmt.Sprintf(configPreamble+`
			//				resource "observe_monitor" "first" {
			//					workspace = data.observe_workspace.default.oid
			//					name      = "%s"
			//
			//					inputs = {
			//						"observation" = data.observe_dataset.observation.oid
			//					}
			//
			//					stage {
			//						pipeline = "filter false"
			//					}
			//
			//					rule {
			//						source_column = "OBSERVATION_KIND"
			//						group_by      = "none"
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
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.observation"),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "stage.0.pipeline"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.count.0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_function", "greater"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_values.0", "100"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.lookback_time", "1m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.baseline_time", "2m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "important"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "separate"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.notification_spec.selection_value"),
			//				),
			//			},
			//			{
			//				Config: fmt.Sprintf(configPreamble+`
			//				resource "observe_monitor" "first" {
			//					workspace = data.observe_workspace.default.oid
			//					name      = "%s"
			//
			//					inputs = {
			//						"observation" = data.observe_dataset.observation.oid
			//					}
			//
			//					stage {
			//						pipeline = "filter false"
			//					}
			//
			//					rule {
			//						source_column = "OBSERVATION_KIND"
			//						group_by      = "none"
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
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.observation"),
			//					resource.TestCheckResourceAttrSet("observe_monitor.first", "stage.0.pipeline"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.source_column", "OBSERVATION_KIND"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.count.0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_function", "greater"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.compare_values.0", "0"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.lookback_time", "1m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.change.0.baseline_time", "2m0s"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "important"),
			//					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "separate"),
			//					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.notification_spec.selection_value"),
			//				),
			//			},
		},
	})
}

func TestAccObserveMonitorThreshold(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {
						pipeline = "colmake temp_number:14"
					}


					rule {
						group_by      = "none"
                        source_column    = "temp_number"

						threshold {
                            compare_function = "greater"
                            compare_values   = [ 70 ]
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
					resource.TestCheckResourceAttrSet("observe_monitor.first", "inputs.observation"),
					resource.TestCheckResourceAttr("observe_monitor.first", "stage.0.pipeline", "colmake temp_number:14"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.compare_values.0", "70"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.threshold.0.lookback_time", "10m0s"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.merge", "merged"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.selection", "any"),
					resource.TestCheckResourceAttr("observe_monitor.first", "notification_spec.0.selection_value", "0"),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "false"),
				),
			},
		},
	})
}

func TestAccObserveMonitorFacet(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {
						pipeline = "filter false"
					}

					rule {
						source_column = "OBSERVATION_KIND"
						group_by      = "none"

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
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_function", "equals"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.facet_values.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.time_function", "never"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.facet.0.lookback_time", "1m0s"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
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
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
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

func TestAccObserveMonitorPromote(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {
						pipeline = <<-EOF
							colmake kind:"test", description:"test"
						EOF
					}

					rule {
						group_by = "resource"

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
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "resource"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", "kind"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", "description"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					disabled  = true

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					rule {
						group_by = "resource"

						promote {
							primary_key       = ["OBSERVATION_KIND"]
						}
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "resource"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
		},
	})
}

func TestAccObserveMonitorGroupByGroup(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					disabled  = true

					inputs = {
						"observation" = data.observe_dataset.observation.oid
					}

					stage {
						pipeline = <<-EOF
							filter true
						EOF
					}

					rule {
						group_by = "value"
						group_by_columns = ["OBSERVATION_KIND"]

						promote {
							primary_key       = ["OBSERVATION_KIND"]
						}
					}

					notification_spec {
						merge       = "separate"
						selection_value = 1
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "value"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_columns.0", "OBSERVATION_KIND"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_monitor" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					disabled  = true

					inputs = {
						"observation" = data.observe_dataset.observation.oid
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
						selection_value = 1
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_monitor.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.columns.0", "OBSERVATION_KIND"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_columns"),
					resource.TestCheckNoResourceAttr("observe_monitor.first", "rule.0.group_by_group.0.name"),
					// previous default lingers, no effect on backend
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.group_by", "none"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.primary_key.#", "1"),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.kind_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "rule.0.promote.0.description_field", ""),
					resource.TestCheckResourceAttr("observe_monitor.first", "disabled", "true"),
				),
			},
		},
	})
}
