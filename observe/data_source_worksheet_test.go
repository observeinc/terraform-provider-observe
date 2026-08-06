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

func TestAccObserveSourceWorksheet_ExportWithBindings(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	// see TestAccObserveSourceDashboard_ExportWithBindings for context on this trick
	providerPreamble := `
		terraform {} # trick the testing framework into not mangling our config
		provider "observe" {
			export_object_bindings = true
		}
	`

	workspaceTfName := fmt.Sprintf("workspace_%s", strings.ToLower(defaultWorkspaceName))
	workspaceTfLocalBindingVar := fmt.Sprintf("binding__worksheet_%s__%s", randomPrefix, workspaceTfName)
	datasetTfName := fmt.Sprintf("worksheet_%s__dataset_%s", randomPrefix, randomPrefix)
	datasetTfLocalBindingVar := fmt.Sprintf("binding__%s", datasetTfName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
					data "observe_oid" "dataset" {
						oid = observe_datastream.test.dataset
					}

					resource "observe_worksheet" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						queries = <<-EOF
						[{
							"pipeline": "filter true",
							"input": [{
								"inputName": "test",
								"inputRole": "Data",
								"datasetId": "${data.observe_oid.dataset.id}"
							}]
						}]
						EOF
					}

					data "observe_worksheet" "lookup" {
						id = observe_worksheet.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_worksheet.lookup", "workspace", fmt.Sprintf("${local.%s}", workspaceTfLocalBindingVar)),
					resource.TestCheckResourceAttrWith("data.observe_worksheet.lookup", "queries", func(val string) error {
						var stagesPartial []struct {
							Input []struct {
								DatasetId string `json:"datasetId"`
							} `json:"input"`
						}
						if err := json.Unmarshal([]byte(val), &stagesPartial); err != nil {
							return err
						}
						expectedId := fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)
						actualId := stagesPartial[0].Input[0].DatasetId
						if actualId != expectedId {
							return fmt.Errorf("expected %#v, got %#v", expectedId, actualId)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.observe_worksheet.lookup", "_bindings", func(value string) error {
						var bindings binding.BindingsObject
						if err := json.Unmarshal([]byte(value), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindDataset, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kinds does not match: expected %#v, got %#v", expectedKinds, bindings.Kinds)
						}
						expectedWorkspaceBinding := binding.Target{TfLocalBindingVar: workspaceTfLocalBindingVar, TfName: workspaceTfName, IsOid: true}
						if bindings.Workspace != expectedWorkspaceBinding {
							return fmt.Errorf("bindings.Workspace does not match: expected %#v, got %#v", expectedWorkspaceBinding, bindings.Workspace)
						}
						expectedDatasetBinding := binding.Target{TfLocalBindingVar: datasetTfLocalBindingVar, TfName: datasetTfName, IsOid: false}
						if b, ok := bindings.Mappings[binding.Ref{Kind: binding.KindDataset, Key: randomPrefix}]; !ok || b != expectedDatasetBinding {
							return fmt.Errorf("bindings.Mappings does not contain expected binding %#v for dataset %s, found: %#v", expectedDatasetBinding, randomPrefix, bindings.Mappings)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccObserveSourceWorksheet(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_worksheet" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%s"
						icon_url  = "test"
						queries = <<-EOF
						[{
							"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
							"input": [{
								"inputName": "kubernetes/metrics/Container Metrics",
								"inputRole": "Data",
								"datasetId": "41042989"
							}]
						}]
						EOF
					}

					data "observe_worksheet" "lookup" {
						workspace = data.observe_workspace.default.oid
						id        = observe_worksheet.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_worksheet.lookup", "name", randomPrefix),
				),
			},
		},
	})
}
