package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

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
