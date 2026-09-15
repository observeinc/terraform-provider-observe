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

// dashboardRestConfig renders a schema_version = 2 dashboard whose content is carried
// entirely by `definition`. The document shape (a layout of titled sections holding
// placed cards) mirrors the content model the dashboards REST API accepts; a single
// self-contained markdown card avoids any dataset dependency. `title` lets a caller
// mutate the definition between steps to exercise the REST PATCH update path.
func dashboardRestConfig(name, title string) string {
	return fmt.Sprintf(`
	resource "observe_dashboard" "rest" {
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

// dashboardRestSectionTitle unmarshals a dashboard definition and returns the title of
// its single layout section, so tests can assert the definition round-trips through the
// REST create/PATCH paths and the GraphQL read.
func dashboardRestSectionTitle(val string) (string, error) {
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

// Verify we can create and update a schema_version >= 2 dashboard. Create routes through
// the REST POST and update routes through the REST PATCH; both are followed by a
// GraphQL read. The intervening PlanOnly steps assert there is no perpetual diff (the
// legacy content fields must stay empty for a new-model dashboard), and the final step
// exercises `terraform import`.
func TestAccObserveDashboardRestCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// schema_version = 2 routes create through the REST API.
				Config: dashboardRestConfig(randomPrefix, "Overview"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.rest", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dashboard.rest", "schema_version", "2"),
					resource.TestCheckResourceAttrSet("observe_dashboard.rest", "definition"),
					resource.TestCheckResourceAttrSet("observe_dashboard.rest", "oid"),
					// Legacy content fields must stay empty for a new-model dashboard.
					resource.TestCheckResourceAttr("observe_dashboard.rest", "stages", ""),
					resource.TestCheckResourceAttrWith("observe_dashboard.rest", "definition", func(val string) error {
						title, err := dashboardRestSectionTitle(val)
						if err != nil {
							return err
						}
						if title != "Overview" {
							return fmt.Errorf("definition did not round-trip: section title = %q, want %q", title, "Overview")
						}
						return nil
					}),
				),
			},
			// Re-applying the same config must not produce drift.
			testAccPlanOnlyNoDriftStep(dashboardRestConfig(randomPrefix, "Overview")),
			{
				// Changing only `definition` routes the update through REST PATCH.
				Config: dashboardRestConfig(randomPrefix, "Overview Updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.rest", "schema_version", "2"),
					resource.TestCheckResourceAttrWith("observe_dashboard.rest", "definition", func(val string) error {
						title, err := dashboardRestSectionTitle(val)
						if err != nil {
							return err
						}
						if title != "Overview Updated" {
							return fmt.Errorf("definition update did not round-trip: section title = %q, want %q", title, "Overview Updated")
						}
						return nil
					}),
				),
			},
			testAccPlanOnlyNoDriftStep(dashboardRestConfig(randomPrefix, "Overview Updated")),
			{
				// Changing only `name` also routes the update through REST PATCH.
				Config: dashboardRestConfig(randomPrefix+"-renamed", "Overview Updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.rest", "name", randomPrefix+"-renamed"),
					resource.TestCheckResourceAttr("observe_dashboard.rest", "schema_version", "2"),
				),
			},
			testAccPlanOnlyNoDriftStep(dashboardRestConfig(randomPrefix+"-renamed", "Overview Updated")),
			{
				ResourceName:      "observe_dashboard.rest",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Verify the legacy/new content-model split is enforced client-side (in CustomizeDiff):
// a schema_version >= 2 dashboard is described entirely by `definition` and must not
// also set the legacy content fields.
func TestAccObserveDashboardRestConflictsWithStages(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "rest" {
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
func TestAccObserveDashboardRestRequiresDefinition(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_dashboard" "rest" {
					name           = "%[1]s"
					schema_version = 2
				}
				`, randomPrefix),
				ExpectError: regexp.MustCompile(`schema_version >= 2 requires 'definition' to be set`),
			},
		},
	})
}

// Verify object_tags create and update for a schema_version >= 2 dashboard, mirroring
// TestAccObserveDashboardObjectTags for the REST create/PATCH paths.
func TestAccObserveDashboardRestObjectTags(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	restConfigWithTags := func(objectTags string) string {
		return fmt.Sprintf(`
		resource "observe_dashboard" "rest" {
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
				Config: restConfigWithTags(`
					team       = "platform"
					visibility = "public,internal" # Will be sorted to "internal,public" by backend
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.rest", "object_tags.team", "platform"),
					resource.TestCheckResourceAttr("observe_dashboard.rest", "object_tags.visibility", "internal,public"), // Backend sorts alphabetically
				),
			},
			{
				// Update object_tags through REST PATCH.
				Config: restConfigWithTags(`
					team = "platform,sre"
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_dashboard.rest", "object_tags.team", "platform,sre"),
					resource.TestCheckNoResourceAttr("observe_dashboard.rest", "object_tags.visibility"),
				),
			},
		},
	})
}
