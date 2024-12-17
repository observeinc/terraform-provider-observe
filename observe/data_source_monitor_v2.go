package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("monitorv2", "description"),
		ReadContext: dataSourceMonitorV2Read,
		Schema: map[string]*schema.Schema{
			// used to lookup the monitor
			// monitor can be looked up either by providing an ID
			// or by providing search params that can uniquely ID the monitor.
			"id": { // ObjectId!
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("common", "schema", "id"),
				ExactlyOneOf:     []string{"name", "id"},
			},
			"workspace": { // ObjectId!
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("monitorv2", "schema", "workspace_id"),
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  descriptions.Get("monitorv2", "schema", "name"),
				ExactlyOneOf: []string{"name", "id"},
			},
			// fields of MonitorV2Input excluding the components of MonitorV2Definition
			"rule_kind": { // MonitorV2RuleKind!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "rule_kind"),
			},
			"icon_url": { // String
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "icon_url"),
			},
			"description": { // String
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "description"),
			},
			"stage": { // for building inputQuery (MultiStageQueryInput!))
				Type: schema.TypeList,
				// we need to declare optional, otherwise we won't get block
				// formatting in state
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"output_stage": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
			"inputs": { // for building inputQuery (MultiStageQueryInput!)
				Type:        schema.TypeMap,
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "inputs"),
			},
			"rules": { // [MonitorV2RuleInput!]!
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "rules", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"level": { // MonitorV2AlarmLevel!
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "rules", "level"),
						},
						"count": { // MonitorV2CountRuleInput
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Description: descriptions.Get("monitorv2", "schema", "rules", "count"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:        schema.TypeList,
										Computed:    true,
										Optional:    true,
										Description: descriptions.Get("monitorv2", "schema", "compare_values"),
										Elem:        monitorV2ComparisonDatasource(),
									},
									"compare_groups": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Computed:    true,
										Optional:    true,
										Description: descriptions.Get("monitorv2", "schema", "compare_groups"),
										Elem:        monitorV2ColumnComparisonDatasource(),
									},
								},
							},
						},
						"threshold": { // MonitorV2ThresholdRuleInput
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "rules", "threshold", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:        schema.TypeList,
										Optional:    true,
										Computed:    true,
										Elem:        monitorV2ComparisonDatasource(),
										Description: descriptions.Get("monitorv2", "schema", "compare_values"),
									},
									"value_column_name": { // String!
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("monitorv2", "schema", "rules", "threshold", "value_column_name"),
									},
									"aggregation": { // MonitorV2ValueAggregation!
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("monitorv2", "schema", "rules", "threshold", "aggregation"),
									},
									"compare_groups": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Optional:    true,
										Computed:    true,
										Elem:        monitorV2ColumnComparisonDatasource(),
										Description: descriptions.Get("monitorv2", "schema", "compare_groups"),
									},
								},
							},
						},
						"promote": { // MonitorV2PromoteRuleInput
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "rules", "promote"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_columns": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Optional:    true,
										Computed:    true,
										Elem:        monitorV2ColumnComparisonDatasource(),
										Description: descriptions.Get("monitorv2", "schema", "column_comparison", "description"),
									},
								},
							},
						},
					},
				},
			},
			"lookback_time": { // Duration
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "lookback_time"),
			},
			"data_stabilization_delay": { // Int64
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "data_stabilization_delay"),
			},
			"max_alerts_per_hour": { //Int64
				Type:        schema.TypeInt,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "max_alerts_per_hour"),
			},
			"groupings": { // [MonitorV2ColumnInput!]
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2ColumnDatasource(),
				Description: descriptions.Get("monitorv2", "schema", "groupings"),
			},
			"scheduling": { // MonitorV2SchedulingInput (required *only* for TF)
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "scheduling", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"transform": { // MonitorV2TransformScheduleInput
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "scheduling", "transform", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"freshness_goal": { // Duration!
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("monitorv2", "schema", "scheduling", "transform", "freshness_goal"),
									},
								},
							},
						},
					},
				},
			},
			"oid": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
			// the following field describes how monitorv2 is connected to shared actions.
			"actions": { // [MonitorV2ActionRuleInput]
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"oid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "actions", "oid"),
						},
						"action": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "actions", "action"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									// fields of MonitorV2ActionInput
									"type": { // MonitorV2ActionType!
										Type:     schema.TypeString,
										Computed: true,
									},
									"email": { // MonitorV2EmailDestinationInput
										Type:     schema.TypeList,
										Computed: true,
										Elem:     monitorV2EmailActionDatasource(),
									},
									"webhook": { // MonitorV2WebhookDestinationInput
										Type:     schema.TypeList,
										Computed: true,
										Elem:     monitorV2WebhookActionDatasource(),
									},
									"description": { // String
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"levels": {
							Type:        schema.TypeList,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: descriptions.Get("monitorv2", "schema", "actions", "levels"),
						},
						"conditions": { // MonitorV2ComparisonExpression
							Type:        schema.TypeList,
							Computed:    true,
							Description: descriptions.Get("monitorv2", "schema", "actions", "conditions", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_terms": { // [MonitorV2ComparisonTerm!]
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"comparison": { // [MonitorV2Comparison!]!
													Type:        schema.TypeList,
													Computed:    true,
													Elem:        monitorV2ComparisonDatasource(),
													Description: descriptions.Get("monitorv2", "schema", "actions", "conditions", "compare_terms", "comparison"),
												},
												"column": { // [MonitorV2Column!]!
													Type:        schema.TypeList,
													Computed:    true,
													Elem:        monitorV2ColumnDatasource(),
													Description: descriptions.Get("monitorv2", "schema", "actions", "conditions", "compare_terms", "column"),
												},
											},
										},
									},
									// note: operator is an implied AND for now until the UI supports OR
								},
							},
						},
						"send_end_notifications": { // Boolean
							Type:     schema.TypeBool,
							Computed: true,
						},
						"send_reminders_interval": { // Duration
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Description: descriptions.Get("monitorv2", "schema", "actions", "description"),
			},
		},
	}
}

func monitorV2ComparisonDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2Comparison
		Schema: map[string]*schema.Schema{
			"compare_fn": { // MonitorV2ComparisonFunction!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "comparison", "compare_fn"),
			},
			"value_int64": { // Int64
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_int64"),
			},
			"value_float64": { // Float
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeFloat},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_float64"),
			},
			"value_bool": { // Boolean
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeBool},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_bool"),
			},
			"value_string": { // String
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_string"),
			},
			"value_duration": { // Int64
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_duration"),
			},
			"value_timestamp": { // Time
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_timestamp"),
			},
		},
	}
}

func monitorV2ColumnPathDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2ColumnPathInput
		Schema: map[string]*schema.Schema{
			"name": { // String!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "column_path", "name"),
			},
			"path": { // String
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "column_path", "path"),
			},
		},
	}
}

func monitorV2LinkColumnDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2LinkColumnInput
		Schema: map[string]*schema.Schema{
			"name": { // String!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitorv2", "schema", "link_column", "name"),
			},
		},
	}
}

func monitorV2ColumnDatasource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"link_column": { // MonitorV2LinkColumnInput
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2LinkColumnDatasource(),
				Description: descriptions.Get("monitorv2", "schema", "link_column", "description"),
			},
			"column_path": { // MonitorV2ColumnPathInput
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2ColumnPathDatasource(),
				Description: descriptions.Get("monitorv2", "schema", "column_path", "description"),
			},
		},
	}
}

func monitorV2ColumnComparisonDatasource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"compare_values": { // [MonitorV2ComparisonInput!]!
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2ComparisonDatasource(),
				Description: descriptions.Get("monitorv2", "schema", "compare_values"),
			},
			"column": { // MonitorV2ColumnInput!
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2ColumnDatasource(),
				Description: descriptions.Get("monitorv2", "schema", "column", "description"),
			},
		},
	}
}

func dataSourceMonitorV2Read(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
		getID  = data.Get("id").(string)
	)

	var m *gql.MonitorV2
	var err error

	if getID != "" {
		m, err = client.GetMonitorV2(ctx, getID)
	} else if name != "" {
		workspaceID, _ := data.Get("workspace").(string)
		if workspaceID != "" {
			m, err = client.LookupMonitorV2(ctx, &workspaceID, &name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	} else if m == nil {
		return diag.Errorf("failed to lookup monitor from provided get/search parameters")
	}

	data.SetId(m.Id)
	return resourceMonitorV2Read(ctx, data, meta)
}
