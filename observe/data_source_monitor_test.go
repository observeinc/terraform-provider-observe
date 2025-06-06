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

func TestAccObserveSourceMonitor(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						disabled  = true

						description = "description"
						comment     = "comment"
						is_template = true
						definition = jsonencode({ "hello" = "world" })

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

						notification_spec {
							importance      = "informational"
							notify_on_close = true
						}
					}

					data "observe_monitor" "lookup" {
						id         = observe_monitor.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "disabled", "true"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "is_template", "true"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "description", "description"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "definition", `{"hello":"world"}`),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "comment", "comment"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						description = "description"
						comment     = "comment"

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}

						rule {
							count {
								compare_function   = "less_or_equal"
								compare_values     = [1]
								lookback_time      = "1m"
							}
						}
					}

					data "observe_monitor" "lookup" {
						id         = observe_monitor.first.id
					}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", "filter false\n"),
				),
			},
		},
	})
}

func TestAccObserveSourceMonitorLookup(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

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

					data "observe_monitor" "lookup" {
						workspace = data.observe_workspace.default.oid
						name      = observe_monitor.first.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"

						inputs = {
							"test" = observe_datastream.test.dataset
						}

						stage {}

						rule {
							source_column = "OBSERVATION_INDEX"

							threshold {
								compare_function = "greater"
								compare_values = [ 75, ]
								lookback_time = "5m0s"
							}
						}

						notification_spec {
							importance      = "informational"
						}
					}

					data "observe_monitor" "lookup" {
						workspace = data.observe_workspace.default.oid
						name      = observe_monitor.first.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "stage.0.pipeline", ""),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "notification_spec.0.importance", "informational"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.threshold.0.compare_function", "greater"),
				),
			},
		},
	})
}

func TestAccObserveSourceMonitorLog(t *testing.T) {
	t.Skip()
	t.Skip()
	t.Skip()
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
				}

				data "observe_monitor" "lookup" {
					id         = observe_monitor.first.id
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.compare_function", "greater"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.compare_values.0", "1"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.lookback_time", "1m0s"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.expression_summary", "Some text"),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "rule.0.log.0.log_stage_id", "stage-0"),
				),
			},
		},
	})
}

func TestAccObserveMonitorExportWithBindings(t *testing.T) {
	randomPrefixMonitor := acctest.RandomWithPrefix("tf")
	randomPrefixDataset := acctest.RandomWithPrefix("tf")

	// see TestAccObserveSourceDashboard_ExportWithBindings for context
	providerPreamble := `
 		terraform {} # trick the testing framework into not mangling our config
 		provider "observe" {
 			export_object_bindings = true
 		}
	`

	workspaceTfName := fmt.Sprintf("workspace_%s", strings.ToLower(defaultWorkspaceName))
	workspaceTfLocalBindingVar := fmt.Sprintf("binding__monitor_%s__%s", randomPrefixMonitor, workspaceTfName)
	datasetTfName := fmt.Sprintf("monitor_%s__dataset_%s", randomPrefixMonitor, randomPrefixDataset)
	datasetTfLocalBindingVar := fmt.Sprintf("binding__%s", datasetTfName)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+monitorConfigPreamble+`
					resource "observe_monitor" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[2]s"
						inputs = {
							"test" = observe_datastream.test.dataset
						}
						stage {
							pipeline = "filter true"
						}
						rule {
							count {
								compare_function   = "less_or_equal"
								compare_values     = [1]
								lookback_time      = "1m"
							}
						}
					}

					data "observe_monitor" "lookup" {
						id = observe_monitor.first.id
					}
				`, randomPrefixDataset, randomPrefixMonitor),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "name", randomPrefixMonitor),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "workspace", fmt.Sprintf("${local.%s}", workspaceTfLocalBindingVar)),
					resource.TestCheckResourceAttr("data.observe_monitor.lookup", "inputs.test", fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)),
					resource.TestCheckResourceAttrWith("data.observe_monitor.lookup", "_bindings", func(val string) error {
						var bindings binding.BindingsObject
						if err := json.Unmarshal([]byte(val), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindDataset, binding.KindWorkspace}
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
						return nil
					}),
				),
			},
		},
	})

}
