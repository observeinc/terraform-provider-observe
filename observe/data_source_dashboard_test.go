package observe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/observeinc/terraform-provider-observe/client/binding"
)

func TestAccObserveSourceDashboard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_dashboard" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%s"
						icon_url  = "test"
						stages = <<-EOF
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

					data "observe_dashboard" "lookup" {
						id        = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dashboard.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
				),
			},
		},
	})
}

func TestAccObserveSourceDashboard_ExportNullParameter(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					data "observe_oid" "dataset" {
						oid = observe_datastream.test.dataset
					}

					resource "observe_dashboard" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						icon_url  = "test"
						stages = <<-EOF
						[{
							"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
							"input": [{
								"inputName": "kubernetes/metrics/Container Metrics",
								"inputRole": "Data",
								"datasetId": "${data.observe_oid.dataset.id}"
							}]
						}]
						EOF

						parameters = jsonencode([
							{
								defaultValue = {
									link = null
								}
								id           = "emptylink"
								name         = "Empty Link"
								valueKind    = {
									type            = "LINK"
									keyForDatasetId = data.observe_oid.dataset.id
								}
							},
						])
					}

					data "observe_dashboard" "lookup" {
						id        = observe_dashboard.first.id
					}

					resource "observe_dashboard" "from_export" {
						workspace  = data.observe_dashboard.lookup.workspace
						name       = "${data.observe_dashboard.lookup.name}-export"
						icon_url   = data.observe_dashboard.lookup.icon_url
						stages     = data.observe_dashboard.lookup.stages
						parameters = data.observe_dashboard.lookup.parameters
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dashboard.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
				),
			},
		},
	})
}

func TestAccObserveSourceDashboard_ExportWithBindings(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	// this is really nasty, but basically if the hashicorp terraform provider testing
	// framework detects a terraform block, it will output the config verbatim instead of
	// trying to insert another resource. their logic is literally `strings.Contains(s.Config, "terraform {")`
	// (hashicorp/terraform-plugin-sdk/v2/helper/resource/teststep_providers.go:24), so
	// there must be a space between the "terraform" and the "{"
	providerPreamble := `
		terraform {} # trick the testing framework into not mangling our config
		provider "observe" {
			export_object_bindings = true
		}
	`
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
					data "observe_oid" "dataset" {
						oid = observe_datastream.test.dataset
					}

					resource "observe_dashboard" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						icon_url  = "test"
						layout    = jsonencode({
							datasetId = data.observe_oid.dataset.id
						})
						stages = <<-EOF
						[{
							"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
							"input": [{
								"inputName": "kubernetes/metrics/Container Metrics",
								"inputRole": "Data",
								"datasetId": "${data.observe_oid.dataset.id}"
							}]
						}]
						EOF

						parameters = jsonencode([
							{
								defaultValue = {
									link = null
								}
								id           = "emptylink"
								name         = "Empty Link"
								valueKind    = {
									type            = "LINK"
									keyForDatasetId = data.observe_oid.dataset.id
								}
							},
						])
					}

					data "observe_dashboard" "lookup" {
						id        = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "layout", func(val string) error {
						// check that we can deserialize a bindings object from the layout
						// field
						var bindings struct {
							DatasetId string                 `json:"datasetId"`
							Bindings  binding.BindingsObject `json:"bindings"`
						}
						if err := json.Unmarshal([]byte(val), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindDataset, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kind does not match: Expected %#v, got %#v", expectedKinds, bindings.Bindings.Kinds)
						}
						expectedId := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						if bindings.DatasetId != expectedId {
							return fmt.Errorf("layout.datasetId does not match: Expected %#v, got %#v", expectedKinds, bindings.Bindings.Kinds)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "stages", func(val string) error {
						var stagesPartial []struct {
							Input []struct {
								DatasetId string `json:"datasetId"`
							} `json:"input"`
						}
						if err := json.Unmarshal([]byte(val), &stagesPartial); err != nil {
							return err
						}
						expectedId := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						actualId := stagesPartial[0].Input[0].DatasetId
						if actualId != expectedId {
							return fmt.Errorf("expected %#v, got %#v", expectedId, actualId)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "parameters", func(val string) error {
						var parametersPartial []struct {
							ValueKind struct {
								KeyForDatasetId string `json:"keyForDatasetId"`
							} `json:"valueKind"`
						}
						if err := json.Unmarshal([]byte(val), &parametersPartial); err != nil {
							return err
						}
						expected_id := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						actual_id := parametersPartial[0].ValueKind.KeyForDatasetId
						if actual_id != expected_id {
							return fmt.Errorf("expected %#v, got %#v", expected_id, actual_id)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccObserveSourceDashboard_ExportWithBindingsEmptyLayout(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	// this is really nasty, but basically if the hashicorp terraform provider testing
	// framework detects a terraform block, it will output the config verbatim instead of
	// trying to insert another resource. their logic is literally `strings.Contains(s.Config, "terraform {")`
	// (hashicorp/terraform-plugin-sdk/v2/helper/resource/teststep_providers.go:24), so
	// there must be a space between the "terraform" and the "{"
	providerPreamble := `
		terraform {} # trick the testing framework into not mangling our config
		provider "observe" {
			export_object_bindings = true
		}
	`
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerPreamble+configPreamble+datastreamConfigPreamble+`
					data "observe_oid" "dataset" {
						oid = observe_datastream.test.dataset
					}

					resource "observe_dashboard" "first" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
						icon_url  = "test"
						stages = <<-EOF
						[{
							"pipeline": "filter field = \"cpu_usage_core_seconds\"\ncolmake cpu_used: value - lag(value, 1), groupby(clusterUid, namespace, podName, containerName)\ncolmake cpu_used: case(\n cpu_used < 0, value, // stream reset for cumulativeCounter metric\n true, cpu_used)\ncoldrop field, value",
							"input": [{
								"inputName": "kubernetes/metrics/Container Metrics",
								"inputRole": "Data",
								"datasetId": "${data.observe_oid.dataset.id}"
							}]
						}]
						EOF

						parameters = jsonencode([
							{
								defaultValue = {
									link = null
								}
								id           = "emptylink"
								name         = "Empty Link"
								valueKind    = {
									type            = "LINK"
									keyForDatasetId = data.observe_oid.dataset.id
								}
							},
						])
					}

					data "observe_dashboard" "lookup" {
						id        = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "layout", func(val string) error {
						// check that we can deserialize a bindings object from the layout
						// field
						var bindings struct {
							DatasetId string                 `json:"datasetId"`
							Bindings  binding.BindingsObject `json:"bindings"`
						}
						if err := json.Unmarshal([]byte(val), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindDataset, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kind does not match: Expected %#v, got %#v", expectedKinds, bindings.Bindings.Kinds)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "stages", func(val string) error {
						var stagesPartial []struct {
							Input []struct {
								DatasetId string `json:"datasetId"`
							} `json:"input"`
						}
						if err := json.Unmarshal([]byte(val), &stagesPartial); err != nil {
							return err
						}
						expectedId := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						actualId := stagesPartial[0].Input[0].DatasetId
						if actualId != expectedId {
							return fmt.Errorf("expected %#v, got %#v", expectedId, actualId)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "parameters", func(val string) error {
						var parametersPartial []struct {
							ValueKind struct {
								KeyForDatasetId string `json:"keyForDatasetId"`
							} `json:"valueKind"`
						}
						if err := json.Unmarshal([]byte(val), &parametersPartial); err != nil {
							return err
						}
						expected_id := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						actual_id := parametersPartial[0].ValueKind.KeyForDatasetId
						if actual_id != expected_id {
							return fmt.Errorf("expected %#v, got %#v", expected_id, actual_id)
						}
						return nil
					}),
				),
			},
		},
	})
}
