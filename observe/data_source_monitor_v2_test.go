package observe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/observeinc/terraform-provider-observe/client/binding"
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
						actions {
							action {
								type = "email"
								email {
									subject = "test operator field"
									body = "testing operator in data source"
									addresses = ["test@observeinc.com"]
									users = [data.observe_user.system.oid]
								}
								description = "test action with conditions"
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
										column_path {
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
										column_path {
											name = "kind"
										}
									}
								}
							}
							send_end_notifications = false
							send_reminders_interval = "15m"
						}
					}

					data "observe_user" "system" {
						email = "%[2]s"
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.id
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "count"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.count.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.count.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.transform.0.freshness_goal", "15m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.action.0.type", "email"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.action.0.description", "test action with conditions"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.levels.0", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.operator", "or"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.0.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.0.comparison.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.0.column.0.column_path.0.name", "description"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.1.comparison.0.compare_fn", "equal"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.1.comparison.0.value_string.0", "test"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.conditions.0.compare_terms.1.column.0.column_path.0.name", "kind"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.send_end_notifications", "false"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.send_reminders_interval", "15m0s"),
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
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "colmake temp_number:14"
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
								compare_values {
									compare_fn    = "greater"
									value_duration = ["1s"]
								}
								value_column_name = "temp_number"
								aggregation = "all_of"
							}
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
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "30m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "threshold"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "no_data_rules.0.expiration", "30m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "no_data_rules.0.threshold.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "no_data_rules.0.threshold.0.aggregation", "all_of"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.0.value_int64.0", "0"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.1.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.compare_values.1.value_duration.0", "1s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.value_column_name", "temp_number"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.threshold.0.aggregation", "all_of"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.transform.0.freshness_goal", "15m0s"),
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
							transform {
								freshness_goal = "15m"
							}
						}
					}

					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "lookback_time", "0s"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rule_kind", "promote"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.level", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.compare_values.0.compare_fn", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.compare_values.0.value_int64.0", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "rules.0.promote.0.compare_columns.0.column.0.column_path.0.name", "temp_number"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "scheduling.0.transform.0.freshness_goal", "15m0s"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2ExportWithBindings(t *testing.T) {
	randomPrefixMonitor := acctest.RandomWithPrefix("tf")
	randomPrefixMonitorAction := acctest.RandomWithPrefix("tf")
	randomPrefixDataset := acctest.RandomWithPrefix("tf")

	// see TestAccObserveSourceDashboard_ExportWithBindings for context
	providerPreamble := `
 		terraform {} # trick the testing framework into not mangling our config
 		provider "observe" {
 			export_object_bindings = true
 		}
	`

	workspaceTfName := fmt.Sprintf("workspace_%s", strings.ToLower(defaultWorkspaceName))
	workspaceTfLocalBindingVar := fmt.Sprintf("binding__monitor_v2_%s__%s", randomPrefixMonitor, workspaceTfName)
	datasetTfName := fmt.Sprintf("monitor_v2_%s__dataset_%s", randomPrefixMonitor, randomPrefixDataset)
	datasetTfLocalBindingVar := fmt.Sprintf("binding__%s", datasetTfName)
	actionTfName := fmt.Sprintf("monitor_v2_%s__monitor_v2_action_%s", randomPrefixMonitor, randomPrefixMonitorAction)
	actionTfLocalBindingVar := fmt.Sprintf("binding__%s", actionTfName)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+monitorV2ConfigPreamble+`
					resource "observe_monitor_v2_action" "action" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "export"
							body = "bindings"
							addresses = ["test@observeinc.com"]
						}
						name = "%[2]s"
					}

					resource "observe_monitor_v2" "first" {
						workspace = data.observe_workspace.default.oid
						rule_kind = "count"
						name = "%[3]s"
						lookback_time = "30m"
						inputs = {
							"test" = observe_datastream.test.dataset
							"test2" = observe_datastream.test.dataset
						}
						stage {
							input = "test"
							pipeline = "filter true"
						}
						stage {
							input = "test2"
							pipeline = "filter true"
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
						actions {
							oid = observe_monitor_v2_action.action.oid
						}
					}
					
					data "observe_monitor_v2" "lookup" {
						id = observe_monitor_v2.first.id
					}
				`, randomPrefixDataset, randomPrefixMonitorAction, randomPrefixMonitor),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "name", randomPrefixMonitor),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "workspace", fmt.Sprintf("${local.%s}", workspaceTfLocalBindingVar)),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "inputs.test", fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "inputs.test2", fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)),
					resource.TestCheckResourceAttr("data.observe_monitor_v2.lookup", "actions.0.oid", fmt.Sprintf("${local.%s}", actionTfLocalBindingVar)),
					resource.TestCheckResourceAttrWith("data.observe_monitor_v2.lookup", "_bindings", func(value string) error {
						var bindings binding.BindingsObject
						if err := json.Unmarshal([]byte(value), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindDataset, binding.KindMonitorV2Action, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kind does not match: Expected %#v, got %#v", expectedKinds, bindings.Kinds)
						}
						expectedWorkspaceBinding := binding.Target{TfLocalBindingVar: workspaceTfLocalBindingVar, TfName: workspaceTfName, IsOid: true}
						if bindings.Workspace != expectedWorkspaceBinding {
							return fmt.Errorf("bindings.Workspace does not match: Expected %#v, got %#v", expectedWorkspaceBinding, bindings.Workspace)
						}
						expectedDatasetBinding := binding.Target{TfLocalBindingVar: datasetTfLocalBindingVar, TfName: datasetTfName, IsOid: true}
						if binding, ok := bindings.Mappings[binding.Ref{Kind: binding.KindDataset, Key: randomPrefixDataset}]; !ok || binding != expectedDatasetBinding {
							return fmt.Errorf("bindings.Mappings does contain expected binding %#v for dataset %s, found bindings: %#v", expectedDatasetBinding, randomPrefixDataset, bindings.Mappings)
						}
						expectedActionBinding := binding.Target{TfLocalBindingVar: actionTfLocalBindingVar, TfName: actionTfName, IsOid: true}
						if binding, ok := bindings.Mappings[binding.Ref{Kind: binding.KindMonitorV2Action, Key: randomPrefixMonitorAction}]; !ok || binding != expectedActionBinding {
							return fmt.Errorf("bindings.Mappings does contain expected binding %#v for action %s, found bindings: %#v", expectedActionBinding, randomPrefixMonitorAction, bindings.Mappings)
						}
						return nil
					}),
				),
			},
		},
	})
}
