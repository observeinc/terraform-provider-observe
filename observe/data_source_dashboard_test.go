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

func TestAccObserveSourceDashboard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_dashboard" "first" {
						workspace   = data.observe_workspace.default.oid
						name        = "%[1]s"
						description = "%[1]s description"
						icon_url    = "test"
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
						id = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "description", randomPrefix+" description"),
					resource.TestCheckResourceAttrSet("data.observe_dashboard.lookup", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "description", randomPrefix+" description"),
				),
			},
		},
	})
}

// Verify the data source reads back the new content-model fields (schema_version and
// definition) for a schema_version >= 2 dashboard.
func TestAccObserveSourceDashboardV2(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_dashboard" "v2" {
						name           = "%[1]s"
						schema_version = 2
						definition = jsonencode({
							layout = {
								sections = [
									{
										title = "Overview"
										cards = []
									},
								]
							}
						})
					}

					data "observe_dashboard" "lookup" {
						id = observe_dashboard.v2.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "schema_version", "2"),
					resource.TestCheckResourceAttrSet("data.observe_dashboard.lookup", "definition"),
					// The legacy content fields are empty for a new-model dashboard.
					resource.TestCheckResourceAttr("data.observe_dashboard.lookup", "stages", ""),
					checkDefinitionSectionTitle("data.observe_dashboard.lookup", "Overview"),
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
	workspaceTfName := fmt.Sprintf("workspace_%s", strings.ToLower(defaultWorkspaceName))
	workspaceTfLocalBindingVar := fmt.Sprintf("binding__dashboard_%s__%s", randomPrefix, workspaceTfName)
	datasetTfName := fmt.Sprintf("dashboard_%s__dataset_%s", randomPrefix, randomPrefix)
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
						expectedWorkspaceBinding := binding.Target{TfLocalBindingVar: workspaceTfLocalBindingVar, TfName: workspaceTfName, IsOid: true}
						if bindings.Bindings.Workspace != expectedWorkspaceBinding {
							return fmt.Errorf("bindings.Workspace does not match: Expected %#v, got %#v", expectedWorkspaceBinding, bindings.Bindings.Workspace)
						}
						expectedDatasetBinding := binding.Target{TfLocalBindingVar: datasetTfLocalBindingVar, TfName: datasetTfName, IsOid: false}
						if binding, ok := bindings.Bindings.Mappings[binding.Ref{Kind: binding.KindDataset, Key: randomPrefix}]; !ok || binding != expectedDatasetBinding {
							return fmt.Errorf("bindings.Mappings does contain expected binding %#v for dataset %s, found bindings: %#v", expectedDatasetBinding, randomPrefix, bindings.Bindings.Mappings)
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
						expectedId := fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)
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
						expected_id := fmt.Sprintf("${local.%s}", datasetTfLocalBindingVar)
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

// dashboardV2InputDatasetID returns the dataset id bound to one card's query input in a
// schema_version >= 2 dashboard definition, identified by its position (section, card,
// input).
func dashboardV2InputDatasetID(val string, sectionIdx, cardIdx, inputIdx int) (string, error) {
	var def struct {
		Layout struct {
			Sections []struct {
				Cards []struct {
					Query struct {
						Content struct {
							Inputs []struct {
								Source struct {
									Dataset struct {
										Id string `json:"id"`
									} `json:"dataset"`
								} `json:"source"`
							} `json:"inputs"`
						} `json:"content"`
					} `json:"query"`
				} `json:"cards"`
			} `json:"sections"`
		} `json:"layout"`
	}
	if err := json.Unmarshal([]byte(val), &def); err != nil {
		return "", fmt.Errorf("failed to parse definition JSON: %w", err)
	}
	if sectionIdx >= len(def.Layout.Sections) {
		return "", fmt.Errorf("expected section %d in definition, got %d sections: %s", sectionIdx, len(def.Layout.Sections), val)
	}
	cards := def.Layout.Sections[sectionIdx].Cards
	if cardIdx >= len(cards) {
		return "", fmt.Errorf("expected card %d in section %d, got %d cards: %s", cardIdx, sectionIdx, len(cards), val)
	}
	inputs := cards[cardIdx].Query.Content.Inputs
	if inputIdx >= len(inputs) {
		return "", fmt.Errorf("expected input %d in card %d, got %d inputs: %s", inputIdx, cardIdx, len(inputs), val)
	}
	return inputs[inputIdx].Source.Dataset.Id, nil
}

// TestAccObserveSourceDashboard_ExportWithBindingsSchemaV2 verifies that cross-tenant
// export bindings are generated for a schema_version >= 2 dashboard's `definition`
// field, the same way they already are for legacy stages/parameters/parameter_values/
// layout.
func TestAccObserveSourceDashboard_ExportWithBindingsSchemaV2(t *testing.T) {
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
						name           = "%[1]s"
						schema_version = 2
						definition = jsonencode({
							layout = {
								sections = [
									{
										title = "Overview"
										cards = [
											{
												type     = "query"
												geometry = { x = 0, y = 0, w = 12, h = 3 }
												query    = {
													content = {
														pipeline = ["filter true"]
														inputs = [
															{
																name   = "test"
																source = {
																	type    = "dataset"
																	dataset = { id = data.observe_oid.dataset.id }
																}
															},
														]
													}
												}
											},
										]
									},
								]
							}
						})
					}

					data "observe_dashboard" "lookup" {
						id = observe_dashboard.first.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.observe_dashboard.lookup", "definition", func(val string) error {
						actualId, err := dashboardV2InputDatasetID(val, 0, 0, 0)
						if err != nil {
							return err
						}
						expectedId := fmt.Sprintf("${local.binding__dashboard_%[1]s__dataset_%[1]s}", randomPrefix)
						if actualId != expectedId {
							return fmt.Errorf("expected definition's dataset id to be replaced with a binding reference: expected %#v, got %#v", expectedId, actualId)
						}
						return nil
					}),
				),
			},
		},
	})
}
