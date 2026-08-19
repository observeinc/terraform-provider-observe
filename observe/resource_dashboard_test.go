package observe

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestValidateDashboardSchemaVersion is a pure unit test (no TF_ACC needed) for the
// alpha warning attached to schema_version: it must emit a warning only when the value
// is >= 2, and must never produce an error diagnostic.
func TestValidateDashboardSchemaVersion(t *testing.T) {
	cases := map[string]struct {
		in       int
		wantWarn bool
	}{
		"unset/zero":  {0, false},
		"v1-explicit": {1, false},
		"v2-alpha":    {2, true},
		"v3":          {3, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			diags := validateDashboardSchemaVersion(tc.in, cty.Path{})
			if diags.HasError() {
				t.Fatalf("schema_version=%d: unexpected error diagnostics: %#v", tc.in, diags)
			}
			gotWarn := false
			for _, d := range diags {
				if d.Severity == diag.Warning {
					gotWarn = true
				}
			}
			if gotWarn != tc.wantWarn {
				t.Fatalf("schema_version=%d: got warning=%v, want %v", tc.in, gotWarn, tc.wantWarn)
			}
		})
	}
}

var (
	dashboardConfigPreamble = `
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
					"datasetId": "41042989"
				}]
			}]
			EOF
		}
		`
)

// Verify we can create dashboards
func TestAccObserveDashboardCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "kubernetes" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-kubernetes"
				}

				locals {
					kubernetes_dataset_id = regex("^o:::dataset:(\\d+)$", observe_datastream.kubernetes.dataset)[0]
				}

				resource "observe_dashboard" "first" {
					workspace        = data.observe_workspace.default.oid
					name             = "%[1]s"
					icon_url         = "test"
					parameter_values = jsonencode(
						[
							{
								id    = "snrk"
								value = {
									string = "value"
								}
							},
						]
					)
					parameters       = jsonencode(
						[
							{
								defaultValue = {
									bool = true
								}
								id           = "onoff"
								name         = "On / Off"
								valueKind    = {
									type = "BOOL"
								}
							},
							{
								defaultValue = {
									float64 = 0.5
								}
								id           = "maybe"
								name         = "Maybe"
								valueKind    = {
									type = "FLOAT64"
								}
							},
						]
					)
					stages = <<-EOF
					[
						{
							"id": "stage-jag28lhh",
							"input": [
							{
								"inputName": "kubernetes/Container Logs",
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 0,
							"label": "Container Logs",
							"steps": [
								{
								"id": "step-idtv2knr",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "kubernetes/Container Logs (41007104)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-eggygj9q",
								"name": "filter (custom)",
								"index": 1,
								"apal": [
									"filter log ~ /\"accounting_collector stats\"/",
									"colmake kvs:parsekvs(log)",
									"coldrop stream, dockerId, containerId, nodeName, log",
									"colmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)",
									"coldrop kvs",
									""
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "filter"
								},
								"columnStatsTable": null,
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "kubernetes/Container Logs",
								"isUserInput": false,
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"linkify": true,
								"loadEverything": false,
								"limit": 1000,
								"stageId": null,
								"resultKinds": [
								"ResultKindStats",
								"ResultKindData",
								"ResultKindSchema",
								"ResultKindProgress"
								],
								"progressive": true,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": "TABLE",
							"appearance": "COLLAPSED",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"kvs": 1164
								},
								"tableHeight": 594,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": false,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "filter log ~ /\"accounting_collector stats\"/\ncolmake kvs:parsekvs(log)\ncoldrop stream, dockerId, containerId, nodeName, log\ncolmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)\ncoldrop kvs\n"
						},
						{
							"id": "stage-obj6v4sw",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 1,
							"label": "Overall Billing SLA",
							"steps": [
								{
								"id": "step-y8v9wdhz",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-qqo3nxnl",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-s6e6z6dm",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 108,
								"SLA": 205,
								"Written": 124,
								"kvs": 1164
								},
								"tableHeight": 110,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
						},
						{
							"id": "stage-06vzzt06",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 2,
							"label": "Per Source Billing SLA",
							"steps": [
								{
								"id": "step-jdt00eo5",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-o2ml8196",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"containerName": "count",
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-8iuuggy5",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 101,
								"SLA": 233,
								"Written": 101,
								"kvs": 1164
								},
								"tableHeight": 179,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
						}
						]
					EOF
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_dashboard.first", "stages"),
				),
			},
		},
	})
}

func TestAccObserveDashboardNullParameterDefaults(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "kubernetes" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-kubernetes"
				}

				locals {
					kubernetes_dataset_id = regex("^o:::dataset:(\\d+)$", observe_datastream.kubernetes.dataset)[0]
				}

				resource "observe_dashboard" "first" {
					workspace        = data.observe_workspace.default.oid
					name             = "%[1]s"
					icon_url         = "test"
					parameter_values = jsonencode(
						[
							{
								id    = "snrk"
								value = {
									string = "value"
								}
							},
						]
					)
					parameters       = jsonencode(
						[
							{
								defaultValue = {
									string = null
								}
								id           = "string"
								name         = "String"
								valueKind    = {
									type = "STRING"
								}
							},
						]
					)
					stages = <<-EOF
					[
						{
							"id": "stage-jag28lhh",
							"input": [
							{
								"inputName": "kubernetes/Container Logs",
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 0,
							"label": "Container Logs",
							"steps": [
								{
								"id": "step-idtv2knr",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "kubernetes/Container Logs (41007104)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-eggygj9q",
								"name": "filter (custom)",
								"index": 1,
								"apal": [
									"filter log ~ /\"accounting_collector stats\"/",
									"colmake kvs:parsekvs(log)",
									"coldrop stream, dockerId, containerId, nodeName, log",
									"colmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)",
									"coldrop kvs",
									""
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "filter"
								},
								"columnStatsTable": null,
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "kubernetes/Container Logs",
								"isUserInput": false,
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"linkify": true,
								"loadEverything": false,
								"limit": 1000,
								"stageId": null,
								"resultKinds": [
								"ResultKindStats",
								"ResultKindData",
								"ResultKindSchema",
								"ResultKindProgress"
								],
								"progressive": true,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": "TABLE",
							"appearance": "COLLAPSED",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"kvs": 1164
								},
								"tableHeight": 594,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": false,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "filter log ~ /\"accounting_collector stats\"/\ncolmake kvs:parsekvs(log)\ncoldrop stream, dockerId, containerId, nodeName, log\ncolmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)\ncoldrop kvs\n"
						},
						{
							"id": "stage-obj6v4sw",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 1,
							"label": "Overall Billing SLA",
							"steps": [
								{
								"id": "step-y8v9wdhz",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-qqo3nxnl",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-s6e6z6dm",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 108,
								"SLA": 205,
								"Written": 124,
								"kvs": 1164
								},
								"tableHeight": 110,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
						},
						{
							"id": "stage-06vzzt06",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 2,
							"label": "Per Source Billing SLA",
							"steps": [
								{
								"id": "step-jdt00eo5",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-o2ml8196",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"containerName": "count",
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-8iuuggy5",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 101,
								"SLA": 233,
								"Written": 101,
								"kvs": 1164
								},
								"tableHeight": 179,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
						}
						]
					EOF
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_dashboard.first", "stages"),
				),
			},
		},
	})
}

func TestAccObserveDashboarIgnoredNullParameterDefaults(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "kubernetes" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-kubernetes"
				}

				locals {
					kubernetes_dataset_id = regex("^o:::dataset:(\\d+)$", observe_datastream.kubernetes.dataset)[0]
				}

				resource "observe_dashboard" "first" {
					workspace        = data.observe_workspace.default.oid
					name             = "%[1]s"
					icon_url         = "test"
					parameter_values = jsonencode(
						[
							{
								id    = "snrk"
								value = {
									string = "value"
								}
							},
						]
					)
					parameters       = jsonencode(
						[
							{
								defaultValue = {
									bool = true
								}
								id           = "onoff"
								name         = "On / Off"
								valueKind    = {
									type = "BOOL"
								}
							},
							{
								defaultValue = {
									bool    = null # should be ignored
									float64 = 0.5
								}
								id           = "maybe"
								name         = "Maybe"
								valueKind    = {
									type = "FLOAT64"
								}
							},
						]
					)
					stages = <<-EOF
					[
						{
							"id": "stage-jag28lhh",
							"input": [
							{
								"inputName": "kubernetes/Container Logs",
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 0,
							"label": "Container Logs",
							"steps": [
								{
								"id": "step-idtv2knr",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "kubernetes/Container Logs (41007104)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-eggygj9q",
								"name": "filter (custom)",
								"index": 1,
								"apal": [
									"filter log ~ /\"accounting_collector stats\"/",
									"colmake kvs:parsekvs(log)",
									"coldrop stream, dockerId, containerId, nodeName, log",
									"colmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)",
									"coldrop kvs",
									""
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "filter"
								},
								"columnStatsTable": null,
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "kubernetes/Container Logs",
								"isUserInput": false,
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"linkify": true,
								"loadEverything": false,
								"limit": 1000,
								"stageId": null,
								"resultKinds": [
								"ResultKindStats",
								"ResultKindData",
								"ResultKindSchema",
								"ResultKindProgress"
								],
								"progressive": true,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": "TABLE",
							"appearance": "COLLAPSED",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"kvs": 1164
								},
								"tableHeight": 594,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": false,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "filter log ~ /\"accounting_collector stats\"/\ncolmake kvs:parsekvs(log)\ncoldrop stream, dockerId, containerId, nodeName, log\ncolmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)\ncoldrop kvs\n"
						},
						{
							"id": "stage-obj6v4sw",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 1,
							"label": "Overall Billing SLA",
							"steps": [
								{
								"id": "step-y8v9wdhz",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-qqo3nxnl",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-s6e6z6dm",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 108,
								"SLA": 205,
								"Written": 124,
								"kvs": 1164
								},
								"tableHeight": 110,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
						},
						{
							"id": "stage-06vzzt06",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 2,
							"label": "Per Source Billing SLA",
							"steps": [
								{
								"id": "step-jdt00eo5",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-o2ml8196",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"containerName": "count",
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-8iuuggy5",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 101,
								"SLA": 233,
								"Written": 101,
								"kvs": 1164
								},
								"tableHeight": 179,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
						}
						]
					EOF
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_dashboard.first", "stages"),
				),
			},
		},
	})
}

// https://observe.atlassian.net/browse/OB-16421
func TestAccObserveDashboard_DefaultValuePrimaryKeyValue(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "kubernetes" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-kubernetes"
				}

				locals {
					kubernetes_dataset_id = regex("^o:::dataset:(\\d+)$", observe_datastream.kubernetes.dataset)[0]
				}

				resource "observe_dashboard" "first" {
					workspace        = data.observe_workspace.default.oid
					name             = "%[1]s"
					icon_url         = "test"
					parameter_values = jsonencode(
						[
							{
								id    = "snrk"
								value = {
									string = "value"
								}
							},
						]
					)
					parameters       = jsonencode(
						[
							{
								defaultValue = {
									bool = true
								}
								id           = "onoff"
								name         = "On / Off"
								valueKind    = {
									type = "BOOL"
								}
							},
							{
								defaultValue = {
									bool    = null # should be ignored
									float64 = 0.5
								}
								id           = "maybe"
								name         = "Maybe"
								valueKind    = {
									type = "FLOAT64"
								}
							},
							{
								defaultValue = {
									link = {
										datasetId = local.kubernetes_dataset_id

										primaryKeyValue = [
											{
												name = "key"
												value = {
													string = "the-value"
												}
											}
										]
									}
								}
								id           = "link"
								name         = "Link"
								valueKind    = {
									keyForDatasetId = local.kubernetes_dataset_id
									type = "LINK"
								}
							},
						]
					)
					stages = <<-EOF
					[
						{
							"id": "stage-jag28lhh",
							"input": [
							{
								"inputName": "kubernetes/Container Logs",
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 0,
							"label": "Container Logs",
							"steps": [
								{
								"id": "step-idtv2knr",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "kubernetes/Container Logs (41007104)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-eggygj9q",
								"name": "filter (custom)",
								"index": 1,
								"apal": [
									"filter log ~ /\"accounting_collector stats\"/",
									"colmake kvs:parsekvs(log)",
									"coldrop stream, dockerId, containerId, nodeName, log",
									"colmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)",
									"coldrop kvs",
									""
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "filter"
								},
								"columnStatsTable": null,
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "kubernetes/Container Logs",
								"isUserInput": false,
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"linkify": true,
								"loadEverything": false,
								"limit": 1000,
								"stageId": null,
								"resultKinds": [
								"ResultKindStats",
								"ResultKindData",
								"ResultKindSchema",
								"ResultKindProgress"
								],
								"progressive": true,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": "TABLE",
							"appearance": "COLLAPSED",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"kvs": 1164
								},
								"tableHeight": 594,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": false,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "filter log ~ /\"accounting_collector stats\"/\ncolmake kvs:parsekvs(log)\ncoldrop stream, dockerId, containerId, nodeName, log\ncolmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)\ncoldrop kvs\n"
						},
						{
							"id": "stage-obj6v4sw",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 1,
							"label": "Overall Billing SLA",
							"steps": [
								{
								"id": "step-y8v9wdhz",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-qqo3nxnl",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-s6e6z6dm",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 108,
								"SLA": 205,
								"Written": 124,
								"kvs": 1164
								},
								"tableHeight": 110,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
						},
						{
							"id": "stage-06vzzt06",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 2,
							"label": "Per Source Billing SLA",
							"steps": [
								{
								"id": "step-jdt00eo5",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-o2ml8196",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"containerName": "count",
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-8iuuggy5",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 101,
								"SLA": 233,
								"Written": 101,
								"kvs": 1164
								},
								"tableHeight": 179,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
						}
						]
					EOF
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_dashboard.first", "stages"),
				),
			},
		},
	})
}

// https://observe.atlassian.net/browse/OB-15881
func TestAccObserveDashboard_DefaultValueInt64(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+dashboardConfigPreamble, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "kubernetes" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-kubernetes"
				}

				locals {
					kubernetes_dataset_id = regex("^o:::dataset:(\\d+)$", observe_datastream.kubernetes.dataset)[0]
				}

				resource "observe_dashboard" "first" {
					workspace        = data.observe_workspace.default.oid
					name             = "%[1]s"
					icon_url         = "test"
					parameter_values = jsonencode(
						[
							{
								id    = "snrk"
								value = {
									string = "value"
								}
							},
						]
					)
					parameters       = jsonencode(
						[
							{
								defaultValue = {
									int64 = "100" # Rendered as strings by the API
								}
								id           = "int"
								name         = "Int"
								valueKind    = {
									type = "INT64"
								}
							},
						]
					)
					stages = <<-EOF
					[
						{
							"id": "stage-jag28lhh",
							"input": [
							{
								"inputName": "kubernetes/Container Logs",
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 0,
							"label": "Container Logs",
							"steps": [
								{
								"id": "step-idtv2knr",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "kubernetes/Container Logs (41007104)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-eggygj9q",
								"name": "filter (custom)",
								"index": 1,
								"apal": [
									"filter log ~ /\"accounting_collector stats\"/",
									"colmake kvs:parsekvs(log)",
									"coldrop stream, dockerId, containerId, nodeName, log",
									"colmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)",
									"coldrop kvs",
									""
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "filter"
								},
								"columnStatsTable": null,
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "kubernetes/Container Logs",
								"isUserInput": false,
								"datasetId": "${local.kubernetes_dataset_id}",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"linkify": true,
								"loadEverything": false,
								"limit": 1000,
								"stageId": null,
								"resultKinds": [
								"ResultKindStats",
								"ResultKindData",
								"ResultKindSchema",
								"ResultKindProgress"
								],
								"progressive": true,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": "TABLE",
							"appearance": "COLLAPSED",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"kvs": 1164
								},
								"tableHeight": 594,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": false,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "filter log ~ /\"accounting_collector stats\"/\ncolmake kvs:parsekvs(log)\ncoldrop stream, dockerId, containerId, nodeName, log\ncolmake Attempted:int64(kvs.num_attempted_collected), Written:int64(kvs.num_written_collected), Failed:int64(kvs.num_failed_collected), Queued:int64(kvs.num_queued_collected), Timedout:int64(kvs.num_queued_collected)\ncoldrop kvs\n"
						},
						{
							"id": "stage-obj6v4sw",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 1,
							"label": "Overall Billing SLA",
							"steps": [
								{
								"id": "step-y8v9wdhz",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-qqo3nxnl",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-s6e6z6dm",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 108,
								"SLA": 205,
								"Written": 124,
								"kvs": 1164
								},
								"tableHeight": 110,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted))"
						},
						{
							"id": "stage-06vzzt06",
							"input": [
							{
								"inputName": "ContainerLogs_0pob",
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
							}
							],
							"layout": {
							"type": "table",
							"index": 2,
							"label": "Per Source Billing SLA",
							"steps": [
								{
								"id": "step-jdt00eo5",
								"name": "Input Step",
								"index": 0,
								"apal": [],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"type": "addDataset"
								},
								"summary": "ContainerLogs_0pob (stage-jag28lhh)",
								"columnStatsTable": null,
								"type": "InputStep",
								"isPinned": false,
								"renderType": null
								},
								{
								"id": "step-o2ml8196",
								"name": "statsby (custom)",
								"index": 1,
								"apal": [
									"statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
								],
								"datasetQuery": null,
								"datasetQueryId": {
									"queryId": null,
									"tableTypes": [
									"TABULAR",
									"SUMMARY"
									],
									"resultKinds": [
									"ResultKindSchema",
									"ResultKindData",
									"ResultKindStats"
									],
									"ignoreCompress": false
								},
								"queryPresentation": {
									"limit": null,
									"stageId": null
								},
								"icon": {
									"iconSet": "remote",
									"type": "math"
								},
								"columnStatsTable": {
									"columnFunctions": {
									"containerName": "count",
									"Ghosts": "count",
									"Timedout": "count",
									"Failed": "count",
									"Queued": "count",
									"Written": "count",
									"Attempted": "count",
									"SLA": "count"
									},
									"datasetQueryId": {
									"queryId": "q-8iuuggy5",
									"tableTypes": [
										"TABULAR"
									],
									"ignoreCompress": false,
									"resultKinds": [
										"ResultKindSchema",
										"ResultKindData"
									]
									}
								},
								"type": "unknown",
								"isPinned": false,
								"renderType": null
								}
							],
							"selectedStepId": null,
							"userInputs": [],
							"systemInputs": [
								{
								"inputName": "ContainerLogs_0pob",
								"isUserInput": false,
								"stageId": "stage-jag28lhh",
								"inputRole": "Data"
								}
							],
							"viewModel": {
								"showTimeRuler": true,
								"scriptTab": "SCRIPT",
								"railCollapseState": {
								"inputsOutputs": false,
								"minimap": false,
								"note": true,
								"script": true
								},
								"stageTab": "table",
								"consoleValue": null,
								"vis": null
							},
							"queryPresentation": {
								"rollup": {},
								"limit": null,
								"stageId": null,
								"initialRollupFilter": {
								"mode": "Last"
								}
							},
							"renderType": null,
							"appearance": "VISIBLE",
							"dataTableViewState": {
								"scrollToColumn": null,
								"scrollToRow": 0,
								"columnWidths": {
								"Attempted": 101,
								"SLA": 233,
								"Written": 101,
								"kvs": 1164
								},
								"tableHeight": 179,
								"autoTableHeight": false,
								"rowHeights": {},
								"rowHeaderWidth": 20,
								"columnHeaderHeight": 29,
								"columnFooterHeight": 0,
								"defaultColumnWidth": 70,
								"hasCalculatedColumnWidths": true,
								"selection": {
								"columns": {},
								"rows": {},
								"cells": {},
								"highlightString": null,
								"columnSelectSequence": [],
								"selectionType": "table"
								},
								"columnVisibility": {},
								"columnOrderOverride": {},
								"summaryColumnVisibility": {},
								"summaryColumnOrderOverride": {},
								"contextMenuXCoord": null,
								"contextMenuYCoord": null,
								"maxColumnWidth": 400,
								"minColumnWidth": 60,
								"minRowHeight": 30,
								"maxMeasuredColumnWidth": {},
								"containerWidth": 1395,
								"tableView": "TABULAR",
								"hasDoneAutoLayout": false,
								"shouldAutoLayout": false,
								"preserveCellAndRowSelection": true,
								"rowSizeIncrement": 1,
								"disableFixedLeftColumns": false,
								"fetchPageSize": 100,
								"eventLinkColumnId": null
							},
							"serializable": true
							},
							"pipeline": "statsby Ghosts:sum(Attempted)-sum(Written), Timedout:sum(Timedout), Failed:sum(Failed), Queued:sum(Queued), Written:sum(Written), Attempted:sum(Attempted), SLA:100*sum(float64(Written))/sum(float64(Attempted)), groupby(containerName)"
						}
						]
					EOF
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_dashboard.first", "stages"),
				),
			},
		},
	})
}

func TestAccObserveDashboardObjectTags(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dashboard" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					icon_url  = "test"
					stages = <<-EOF
					[{
						"pipeline": "filter field = \"cpu_usage_core_seconds\"",
						"input": [{
							"inputName": "kubernetes/metrics/Container Metrics",
							"inputRole": "Data",
							"datasetId": "41042989"
						}]
					}]
					EOF

					object_tags = {
						team       = "platform"
						visibility = "public,internal"  # Will be sorted to "internal,public" by backend
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.first", "object_tags.team", "platform"),
					resource.TestCheckResourceAttr("observe_dashboard.first", "object_tags.visibility", "internal,public"), // Backend sorts alphabetically
				),
			},
			{
				// Update object_tags
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_dashboard" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					icon_url  = "test"
					stages = <<-EOF
					[{
						"pipeline": "filter field = \"cpu_usage_core_seconds\"",
						"input": [{
							"inputName": "kubernetes/metrics/Container Metrics",
							"inputRole": "Data",
							"datasetId": "41042989"
						}]
					}]
					EOF

					object_tags = {
						team = "platform,sre"
					}
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.first", "object_tags.team", "platform,sre"),
					resource.TestCheckNoResourceAttr("observe_dashboard.first", "object_tags.visibility"),
				),
			},
		},
	})
}

// Verify that dashboards whose parameters reference a correlation tag survive
// the Read path used by both `terraform refresh` and `terraform import`. The
// Dashboard GraphQL fragment must request `valueKind.tagName`; otherwise it is
// silently dropped on read, `diffSuppressParameters` (which unmarshals into
// []ParameterSpecInput and compares with cmp.Equal) sees a mismatch against the
// user's HCL, and Terraform reports a perpetual diff.
func TestAccObserveDashboardImport_CorrelationTagParameter(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	expectedTagName := randomPrefix + "-tag"

	dashboardConfig := fmt.Sprintf(linkConfigPreamble+`
		resource "observe_correlation_tag" "ctag" {
			name    = "%[1]s-tag"
			dataset = observe_dataset.a.oid
			column  = "key"
		}

		resource "observe_dashboard" "with_correlation_tag" {
			workspace  = data.observe_workspace.default.oid
			name       = "%[1]s"
			icon_url   = "test"
			depends_on = [observe_correlation_tag.ctag]

			stages = <<-EOF
			[{
				"pipeline": "filter field = \"cpu_usage_core_seconds\"",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF

			parameters = jsonencode([
				{
					id        = "ctag_param"
					name      = "Correlation Tag Param"
					valueKind = {
						type    = "CORRELATION_TAG"
						tagName = "%[1]s-tag"
					}
				},
			])
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: dashboardConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.with_correlation_tag", "name", randomPrefix),
					resource.TestCheckResourceAttrWith(
						"observe_dashboard.with_correlation_tag",
						"parameters",
						func(val string) error {
							var params []struct {
								Id        string `json:"id"`
								ValueKind struct {
									Type    string  `json:"type"`
									TagName *string `json:"tagName"`
								} `json:"valueKind"`
							}
							if err := json.Unmarshal([]byte(val), &params); err != nil {
								return fmt.Errorf("failed to parse parameters JSON: %w", err)
							}
							if len(params) != 1 {
								return fmt.Errorf("expected 1 parameter, got %d", len(params))
							}
							if got, want := params[0].ValueKind.Type, "CORRELATION_TAG"; got != want {
								return fmt.Errorf("valueKind.type = %q, want %q", got, want)
							}
							if params[0].ValueKind.TagName == nil {
								return fmt.Errorf("valueKind.tagName was dropped on read; expected %q", expectedTagName)
							}
							if got := *params[0].ValueKind.TagName; got != expectedTagName {
								return fmt.Errorf("valueKind.tagName = %q, want %q", got, expectedTagName)
							}
							return nil
						},
					),
				),
			},
			// Re-applying the same config must not produce drift.
			{
				Config:             dashboardConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Exercise `terraform import` directly. ImportStateVerify compares
			// the imported state against the post-create state; if either Read
			// path is incomplete the comparison will fail.
			{
				ResourceName:      "observe_dashboard.with_correlation_tag",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// dashboardExtractV2SectionTitle unmarshals a dashboard definition and returns the title of
// its single layout section, so tests can assert the definition round-trips through the
// REST create/PATCH paths and the GraphQL read.
func dashboardExtractV2SectionTitle(val string) (string, error) {
	var def struct {
		Layout struct {
			Sections []struct {
				Title string `json:"title"`
			} `json:"sections"`
		} `json:"layout"`
	}
	if err := json.Unmarshal([]byte(val), &def); err != nil {
		return "", fmt.Errorf("failed to parse definition JSON: %w", err)
	}
	if len(def.Layout.Sections) != 1 {
		return "", fmt.Errorf("expected 1 section in definition, got %d: %s", len(def.Layout.Sections), val)
	}
	return def.Layout.Sections[0].Title, nil
}

// checkDefinitionSectionTitle asserts resourceAddr's `definition` round-trips with a
// single layout section titled wantTitle.
func checkDefinitionSectionTitle(resourceAddr, wantTitle string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceAddr, "definition", func(val string) error {
		title, err := dashboardExtractV2SectionTitle(val)
		if err != nil {
			return err
		}
		if title != wantTitle {
			return fmt.Errorf("definition did not round-trip: section title = %q, want %q", title, wantTitle)
		}
		return nil
	})
}

// Verify we can create and update a schema_version >= 2 dashboard. Create routes through
// the REST POST and update routes through the REST PATCH; both are followed by a
// GraphQL read. The intervening PlanOnly steps assert there is no perpetual diff (the
// legacy content fields must stay empty for a new-model dashboard), and the final step
// exercises `terraform import`.
func TestAccObserveDashboardV2Create(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	// A single self-contained markdown card avoids any dataset dependency; title lets
	// each step mutate the definition to exercise the REST PATCH update path.
	dashboardV2Config := func(name, title string) string {
		return fmt.Sprintf(`
		resource "observe_dashboard" "v2" {
			name           = "%[1]s"
			description    = "%[1]s description"
			schema_version = 2
			definition = jsonencode({
				layout = {
					sections = [
						{
							title = "%[2]s"
							cards = [
								{
									type     = "markdown"
									geometry = { x = 0, y = 0, w = 12, h = 3 }
									markdown = { title = "Welcome", body = "# %[1]s\n\nManaged by Terraform." }
								},
							]
						},
					]
				}
			})
		}
		`, name, title)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// schema_version = 2 routes create through the REST API.
				Config: dashboardV2Config(randomPrefix, "Overview"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.v2", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.v2", "schema_version", "2"),
					resource.TestCheckResourceAttrSet("observe_dashboard.v2", "definition"),
					resource.TestCheckResourceAttrSet("observe_dashboard.v2", "oid"),
					// Legacy content fields must stay empty for a new-model dashboard.
					resource.TestCheckResourceAttr("observe_dashboard.v2", "stages", ""),
					checkDefinitionSectionTitle("observe_dashboard.v2", "Overview"),
				),
			},
			// Re-applying the same config must not produce drift.
			testAccPlanOnlyNoDriftStep(dashboardV2Config(randomPrefix, "Overview")),
			{
				// Changing only `definition` routes the update through REST PATCH.
				Config: dashboardV2Config(randomPrefix, "Overview Updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.v2", "schema_version", "2"),
					checkDefinitionSectionTitle("observe_dashboard.v2", "Overview Updated"),
				),
			},
			testAccPlanOnlyNoDriftStep(dashboardV2Config(randomPrefix, "Overview Updated")),
			{
				// Changing only `name` also routes the update through REST PATCH.
				Config: dashboardV2Config(randomPrefix+"-renamed", "Overview Updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.v2", "name", randomPrefix+"-renamed"),
					resource.TestCheckResourceAttr("observe_dashboard.v2", "schema_version", "2"),
				),
			},
			testAccPlanOnlyNoDriftStep(dashboardV2Config(randomPrefix+"-renamed", "Overview Updated")),
			{
				ResourceName:      "observe_dashboard.v2",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Verify the legacy/new content-model split is enforced client-side (in CustomizeDiff):
// a schema_version >= 2 dashboard is described entirely by `definition` and must not
// also set the legacy content fields.
func TestAccObserveDashboardV2ConflictsWithStages(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "v2" {
					name           = "%[1]s"
					schema_version = 2
					definition     = jsonencode({ layout = { sections = [] } })
					stages = <<-EOF
					[{
						"pipeline": "filter field = \"cpu_usage_core_seconds\"",
						"input": [{
							"inputName": "kubernetes/metrics/Container Metrics",
							"inputRole": "Data",
							"datasetId": "41042989"
						}]
					}]
					EOF
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`'stages' must not be set when schema_version >= 2`),
			},
		},
	})
}

// Verify that schema_version >= 2 requires `definition` to be set.
func TestAccObserveDashboardV2RequiresDefinition(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "v2" {
					name           = "%[1]s"
					schema_version = 2
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`schema_version >= 2 requires 'definition' to be set`),
			},
		},
	})
}

// Verify the legacy/new content-model split is enforced client-side (in
// CustomizeDiff) for the legacy branch too: a schema_version < 2 dashboard must not
// set `definition` -- the mirror image of TestAccObserveDashboardV2ConflictsWithStages.
func TestAccObserveDashboardLegacyConflictsWithDefinition(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "legacy" {
					name       = "%[1]s"
					definition = jsonencode({ layout = { sections = [] } })
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`'definition' can only be set when schema_version >= 2`),
			},
		},
	})
}

// Verify that a legacy dashboard (schema_version < 2) requires `stages` to be set --
// the mirror image of TestAccObserveDashboardV2RequiresDefinition. This became
// load-bearing once `stages` changed from Required to Optional in the schema: the SDK
// no longer enforces it, so this CustomizeDiff check is now the only thing that does.
func TestAccObserveDashboardLegacyRequiresStages(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "legacy" {
					name = "%[1]s"
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`'stages' is required for legacy dashboards`),
			},
		},
	})
}

// Verify object_tags create and update for a schema_version >= 2 dashboard, mirroring
// TestAccObserveDashboardObjectTags for the REST create/PATCH paths.
func TestAccObserveDashboardV2ObjectTags(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	v2ConfigWithTags := func(objectTags string) string {
		return fmt.Sprintf(`
		resource "observe_dashboard" "v2" {
			name           = "%[1]s"
			schema_version = 2
			definition = jsonencode({
				layout = {
					sections = []
				}
			})
			object_tags = {
				%[2]s
			}
		}
		`, randomPrefix, objectTags)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: v2ConfigWithTags(`
					team       = "platform"
					visibility = "public,internal" # Will be sorted to "internal,public" by backend
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.v2", "object_tags.team", "platform"),
					resource.TestCheckResourceAttr("observe_dashboard.v2", "object_tags.visibility", "internal,public"), // Backend sorts alphabetically
				),
			},
			{
				// Update object_tags through REST PATCH.
				Config: v2ConfigWithTags(`
					team = "platform,sre"
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.v2", "object_tags.team", "platform,sre"),
					resource.TestCheckNoResourceAttr("observe_dashboard.v2", "object_tags.visibility"),
				),
			},
		},
	})
}

// TestAccObserveDashboardLegacyToV2 verifies a dashboard can move between the
// legacy (schema_version < 2) and v2 (schema_version >= 2) content models, in
// both directions, via a plain terraform apply -- always updated in place,
// never destroyed and recreated.
func TestAccObserveDashboardLegacyToV2(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	var dashboardID string

	// Same object across every step, proving each transition updated the
	// dashboard in place rather than destroying and recreating it.
	checkSameID := resource.TestCheckResourceAttrWith("observe_dashboard.test", "id", func(v string) error {
		if dashboardID == "" {
			if v == "" {
				return fmt.Errorf("expected a dashboard id to be set")
			}
			dashboardID = v
			return nil
		}
		if v != dashboardID {
			return fmt.Errorf("expected dashboard to be updated in place, but id changed from %q to %q", dashboardID, v)
		}
		return nil
	})

	// Same OPAL content on both sides of the transition, so the legacy stage and the v2
	// query card are genuinely equivalent dashboards, not two unrelated skeletons.
	const pipeline = "timechart options(empty_bins:true), count: count(1), group_by()"

	// The datastream (and its dataset) is declared identically in every step's config
	// below, so it's never destroyed/recreated across transitions, and both the legacy
	// stage and the v2 card query the same dataset throughout.
	preamble := fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
		data "observe_oid" "dataset" {
			oid = observe_datastream.test.dataset
		}
	`, randomPrefix)

	legacyConfig := preamble + fmt.Sprintf(`
		resource "observe_dashboard" "test" {
			workspace   = data.observe_workspace.default.oid
			name        = "%[1]s"
			description = "%[1]s description"
			icon_url    = "test"
			stages = jsonencode(
				[
					{
						id     = "stage-nkeju1il"
						params = null
						input = [
							{
								datasetId   = data.observe_oid.dataset.id
								datasetPath = null
								inputName   = "test"
								inputRole   = "Data"
								stageId     = null
							},
						]
						pipeline = %[2]q
					},
				]
			)
			parameters = jsonencode([
				{
					id           = "greeting"
					name         = "Greeting"
					valueKind    = { type = "STRING" }
					defaultValue = { string = "hello" }
				},
			])
			parameter_values = jsonencode([
				{
					id    = "greeting"
					value = { string = "world" }
				},
			])
			# A real legacy dashboard always has one of these -- the UI writes it on
			# every save, even for a single unplaced card -- so a bare stage with no
			# layout at all isn't representative.
			layout = jsonencode({
				autoPack = true
				gridLayout = {
					sections = [
						{
							card = {
								title    = "Dashboard content"
								closed   = false
								cardType = "section"
							}
							items = [
								{
									card = {
										stageId  = "stage-nkeju1il"
										cardType = "stage"
									}
									layout = { h = 12, w = 4, x = 0, y = 0 }
								},
							]
						},
					]
				}
			})
			object_tags = {
				team = "platform"
			}
		}
	`, randomPrefix, pipeline)

	// icon_url and stages are omitted: dashboardCustomizeDiff forbids setting them
	// once schema_version >= 2. description and object_tags carry over unchanged, and
	// the query card carries the same dataset+pipeline as the legacy stage above.
	v2Config := func(title string) string {
		return preamble + fmt.Sprintf(`
			resource "observe_dashboard" "test" {
				name           = "%[1]s"
				description    = "%[1]s description"
				schema_version = 2
				definition = jsonencode({
					layout = {
						sections = [
							{
								title = "%[2]s"
								cards = [
									{
										type     = "query"
										geometry = { x = 0, y = 0, w = 12, h = 6 }
										query = {
											content = {
												pipeline = [%[3]q]
												inputs = [
													{
														name = "test"
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
				object_tags = {
					team = "platform"
				}
			}
		`, randomPrefix, title, pipeline)
	}

	// The dashboard's stage/card always queries the same dataset; checkDatasetID
	// captures its ID on first sight and requires every later step to match it, the
	// same "captured once, compared every step" pattern as checkSameID above.
	var datasetID string
	checkDatasetID := func(id string) error {
		if id == "" {
			return fmt.Errorf("expected a dataset id to be set")
		}
		if datasetID == "" {
			datasetID = id
			return nil
		}
		if id != datasetID {
			return fmt.Errorf("expected the dashboard to keep querying the same dataset, but id changed from %q to %q", datasetID, id)
		}
		return nil
	}

	// checkLegacy/checkV2 assert every field, so a step's Check catches drift anywhere --
	// not just in whatever field that step happens to be exercising. Each attribute's
	// parsing and round-trip comparison happen together in one closure, since each is
	// only ever checked in this one place.
	checkLegacy := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("observe_dashboard.test", "name", randomPrefix),
		resource.TestCheckResourceAttr("observe_dashboard.test", "description", randomPrefix+" description"),
		resource.TestCheckResourceAttr("observe_dashboard.test", "schema_version", "0"),
		resource.TestCheckResourceAttr("observe_dashboard.test", "icon_url", "test"),
		resource.TestCheckResourceAttr("observe_dashboard.test", "object_tags.team", "platform"),
		resource.TestCheckResourceAttrSet("observe_dashboard.test", "oid"),
		resource.TestCheckResourceAttr("observe_dashboard.test", "definition", ""),
		resource.TestCheckResourceAttrWith("observe_dashboard.test", "stages", func(val string) error {
			var stages []struct {
				Pipeline string `json:"pipeline"`
				Input    []struct {
					DatasetId string `json:"datasetId"`
				} `json:"input"`
			}
			if err := json.Unmarshal([]byte(val), &stages); err != nil {
				return fmt.Errorf("failed to parse stages JSON: %w", err)
			}
			if len(stages) != 1 || len(stages[0].Input) != 1 {
				return fmt.Errorf("expected exactly 1 stage with 1 input, got: %s", val)
			}
			if stages[0].Pipeline != pipeline {
				return fmt.Errorf("stages did not round-trip: pipeline = %q, want %q", stages[0].Pipeline, pipeline)
			}
			return checkDatasetID(stages[0].Input[0].DatasetId)
		}),
		resource.TestCheckResourceAttrWith("observe_dashboard.test", "layout", func(val string) error {
			var layout struct {
				GridLayout struct {
					Sections []struct {
						Items []struct {
							Card struct {
								StageId string `json:"stageId"`
							} `json:"card"`
						} `json:"items"`
					} `json:"sections"`
				} `json:"gridLayout"`
			}
			if err := json.Unmarshal([]byte(val), &layout); err != nil {
				return fmt.Errorf("failed to parse layout JSON: %w", err)
			}
			if len(layout.GridLayout.Sections) != 1 || len(layout.GridLayout.Sections[0].Items) != 1 {
				return fmt.Errorf("expected exactly 1 section with 1 item, got: %s", val)
			}
			if stageID := layout.GridLayout.Sections[0].Items[0].Card.StageId; stageID != "stage-nkeju1il" {
				return fmt.Errorf("layout did not round-trip: stageId = %q, want %q", stageID, "stage-nkeju1il")
			}
			return nil
		}),
		resource.TestCheckResourceAttrWith("observe_dashboard.test", "parameters", func(val string) error {
			var params []struct {
				Id           string `json:"id"`
				DefaultValue struct {
					String string `json:"string"`
				} `json:"defaultValue"`
			}
			if err := json.Unmarshal([]byte(val), &params); err != nil {
				return fmt.Errorf("failed to parse parameters JSON: %w", err)
			}
			if len(params) != 1 {
				return fmt.Errorf("expected exactly 1 parameter, got: %s", val)
			}
			if params[0].Id != "greeting" || params[0].DefaultValue.String != "hello" {
				return fmt.Errorf("parameters did not round-trip: got id=%q defaultValue=%q, want id=%q defaultValue=%q", params[0].Id, params[0].DefaultValue.String, "greeting", "hello")
			}
			return nil
		}),
		resource.TestCheckResourceAttrWith("observe_dashboard.test", "parameter_values", func(val string) error {
			var values []struct {
				Id    string `json:"id"`
				Value struct {
					String string `json:"string"`
				} `json:"value"`
			}
			if err := json.Unmarshal([]byte(val), &values); err != nil {
				return fmt.Errorf("failed to parse parameter_values JSON: %w", err)
			}
			if len(values) != 1 {
				return fmt.Errorf("expected exactly 1 parameter value, got: %s", val)
			}
			if values[0].Id != "greeting" || values[0].Value.String != "world" {
				return fmt.Errorf("parameter_values did not round-trip: got id=%q value=%q, want id=%q value=%q", values[0].Id, values[0].Value.String, "greeting", "world")
			}
			return nil
		}),
		checkSameID,
	)
	checkV2 := func(definitionTitle string) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("observe_dashboard.test", "name", randomPrefix),
			resource.TestCheckResourceAttr("observe_dashboard.test", "description", randomPrefix+" description"),
			resource.TestCheckResourceAttr("observe_dashboard.test", "schema_version", "2"),
			resource.TestCheckResourceAttr("observe_dashboard.test", "icon_url", ""),
			resource.TestCheckResourceAttr("observe_dashboard.test", "object_tags.team", "platform"),
			resource.TestCheckResourceAttrSet("observe_dashboard.test", "oid"),
			resource.TestCheckResourceAttr("observe_dashboard.test", "stages", ""),
			resource.TestCheckResourceAttr("observe_dashboard.test", "layout", ""),
			resource.TestCheckResourceAttr("observe_dashboard.test", "parameters", ""),
			resource.TestCheckResourceAttr("observe_dashboard.test", "parameter_values", ""),
			resource.TestCheckResourceAttrWith("observe_dashboard.test", "definition", func(val string) error {
				title, err := dashboardExtractV2SectionTitle(val)
				if err != nil {
					return err
				}
				if title != definitionTitle {
					return fmt.Errorf("definition did not round-trip: section title = %q, want %q", title, definitionTitle)
				}
				var def struct {
					Layout struct {
						Sections []struct {
							Cards []struct {
								Query struct {
									Content struct {
										Pipeline []string `json:"pipeline"`
										Inputs   []struct {
											Source struct {
												Dataset struct {
													ID string `json:"id"`
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
					return fmt.Errorf("failed to parse definition JSON: %w", err)
				}
				if len(def.Layout.Sections) != 1 || len(def.Layout.Sections[0].Cards) != 1 {
					return fmt.Errorf("expected exactly 1 section with 1 card, got: %s", val)
				}
				content := def.Layout.Sections[0].Cards[0].Query.Content
				if len(content.Inputs) != 1 {
					return fmt.Errorf("expected exactly 1 input, got: %s", val)
				}
				if len(content.Pipeline) != 1 || content.Pipeline[0] != pipeline {
					return fmt.Errorf("definition did not round-trip: card pipeline = %v, want [%q]", content.Pipeline, pipeline)
				}
				return checkDatasetID(content.Inputs[0].Source.Dataset.ID)
			}),
			checkSameID,
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Start out as a legacy (stages-based) dashboard.
				Config: legacyConfig,
				Check:  checkLegacy,
			},
			testAccPlanOnlyNoDriftStep(legacyConfig),
			{
				// Transition the same resource in place to the v2 model.
				Config: v2Config("Overview"),
				Check:  checkV2("Overview"),
			},
			testAccPlanOnlyNoDriftStep(v2Config("Overview")),
			{
				// Update again while staying on schema_version = 2: the REST PATCH
				// (merge-patch) path for an already-v2 dashboard.
				Config: v2Config("Overview Updated"),
				Check:  checkV2("Overview Updated"),
			},
			testAccPlanOnlyNoDriftStep(v2Config("Overview Updated")),
			{
				// Switch back to legacy: the reverse transition.
				Config: legacyConfig,
				Check:  checkLegacy,
			},
			testAccPlanOnlyNoDriftStep(legacyConfig),
		},
	})
}
