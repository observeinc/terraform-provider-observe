package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func datasourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("monitorv2", "description"),
		ReadContext: dataSourceMonitorV2Read,
		Schema: map[string]*schema.Schema{
			// needed as input to MonitorV2Create, also part of MonitorV2 struct
			"workspace_id": { // ObjectId!
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("monitorv2", "schema", "workspace_id"),
			},
			// fields of MonitorV2Input excluding the components of MonitorV2DefinitionInput
			"comment": { // String
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "comment"),
			},
			"rule_kind": { // MonitorV2RuleKind!
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2RuleKinds),
				Description:      descriptions.Get("monitorv2", "schema", "rule_kind"),
			},
			"name": { // String!
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "name"),
			},
			"icon_url": { // String
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "icon_url"),
			},
			"description": { // String
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "description"),
			},
			// until specified otherwise, the following are for building MonitorV2DefinitionInput
			"stage": { // for building inputQuery (MultiStageQueryInput!))
				Type:        schema.TypeList,
				MinItems:    1,
				Required:    true,
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:     schema.TypeString,
							Optional: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// ignore alias for last stage, because it won't be set anyway
								stage := d.Get("stage").([]interface{})
								return k == fmt.Sprintf("stage.%d.alias", len(stage)-1)
							},
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"output_stage": {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
			"inputs": { // for building inputQuery (MultiStageQueryInput!)
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
				Description:      descriptions.Get("transform", "schema", "inputs"),
			},
			"rules": { // [MonitorV2RuleInput!]!
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: descriptions.Get("monitorv2", "schema", "rules", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"level": { // MonitorV2AlarmLevel!
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateEnums(gql.AllMonitorV2AlarmLevels),
							Description:      descriptions.Get("monitorv2", "schema", "rules", "level"),
						},
						"count": { // MonitorV2CountRuleInput
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: descriptions.Get("monitorv2", "schema", "rules", "count"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:        schema.TypeList,
										Required:    true,
										MinItems:    1,
										Description: descriptions.Get("monitorv2", "schema", "compare_values"),
										Elem:        monitorV2ComparisonResource(),
									},
									"compare_groups": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Optional:    true,
										Description: descriptions.Get("monitorv2", "schema", "compare_groups"),
										Elem:        monitorV2ColumnComparisonResource(),
									},
								},
							},
						},
						"threshold": { // MonitorV2ThresholdRuleInput
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: descriptions.Get("monitorv2", "schema", "rules", "threshold", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:        schema.TypeList,
										Required:    true,
										MinItems:    1,
										Elem:        monitorV2ComparisonResource(),
										Description: descriptions.Get("monitorv2", "schema", "compare_values"),
									},
									"value_column_name": { // String!
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("monitorv2", "schema", "rules", "threshold", "value_column_name"),
									},
									"aggregation": { // MonitorV2ValueAggregation!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllMonitorV2ValueAggregations),
										Description:      descriptions.Get("monitorv2", "schema", "rules", "threshold", "aggregation"),
									},
									"compare_groups": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Optional:    true,
										Elem:        monitorV2ColumnComparisonResource(),
										Description: descriptions.Get("monitorv2", "schema", "compare_groups"),
									},
								},
							},
						},
						"promote": { // MonitorV2PromoteRuleInput
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: descriptions.Get("monitorv2", "schema", "rules", "promote"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_columns": { // [MonitorV2ColumnComparisonInput!]
										Type:        schema.TypeList,
										Optional:    true,
										Elem:        monitorV2ColumnComparisonResource(),
										Description: descriptions.Get("monitorv2", "schema", "column_comparison", "description"),
									},
								},
							},
						},
					},
				},
			},
			"lookback_time": { // Duration
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
				Description:      descriptions.Get("monitorv2", "schema", "lookback_time"),
			},
			"data_stabilization_delay": { // Duration
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
				Description:      descriptions.Get("monitorv2", "schema", "data_stabilization_delay"),
			},
			"groupings": { // [MonitorV2ColumnInput!]
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        monitorV2ColumnResource(),
				Description: descriptions.Get("monitorv2", "schema", "groupings"),
			},
			"scheduling": { // MonitorV2SchedulingInput (required *only* for TF)
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: descriptions.Get("monitorv2", "schema", "scheduling", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": { // MonitorV2IntervalScheduleInput
							Type:         schema.TypeList,
							Optional:     true,
							MaxItems:     1,
							ExactlyOneOf: []string{"scheduling.0.interval", "scheduling.0.transform"},
							Description:  descriptions.Get("monitorv2", "schema", "scheduling", "interval", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"interval": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
										Description:      descriptions.Get("monitorv2", "schema", "scheduling", "interval", "interval"),
									},
									"randomize": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
										Description:      descriptions.Get("monitorv2", "schema", "scheduling", "interval", "randomize"),
									},
								},
							},
						},
						"transform": { // MonitorV2TransformScheduleInput
							Type:         schema.TypeList,
							Optional:     true,
							MaxItems:     1,
							ExactlyOneOf: []string{"scheduling.0.interval", "scheduling.0.transform"},
							Description:  descriptions.Get("monitorv2", "schema", "scheduling", "transform", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"freshness_goal": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
										Description:      descriptions.Get("monitorv2", "schema", "scheduling", "transform", "freshness_goal"),
									},
								},
							},
						},
					},
				},
			},
			// end of fields of MonitorV2DefinitionInput
			// the following fields are those that aren't given as input to CU ops, but can be read by R ops.
			"id": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2ComparisonDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2Comparison
		Schema: map[string]*schema.Schema{
			"compare_fn": { // MonitorV2ComparisonFunction!
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
				Description:      descriptions.Get("monitorv2", "schema", "comparison", "compare_fn"),
			},
			"value_int64": { // Int64
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_int64"),
			},
			"value_float64": { // Float
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeFloat},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_float64"),
			},
			"value_bool": { // Boolean
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeBool},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_bool"),
			},
			"value_string": { // String
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_string"),
			},
			"value_duration": { // Int64
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Schema{
					Type:             schema.TypeBool,
					ValidateDiagFunc: validateTimeDuration,
					DiffSuppressFunc: diffSuppressTimeDurationZeroDistinctFromEmpty,
				},
				Description: descriptions.Get("monitorv2", "schema", "comparison", "value_duration"),
			},
			"value_timestamp": { // Time
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateTimestamp,
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
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "column_path", "name"),
			},
			"path": { // String
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "column_path", "path"),
			},
		},
	}
}

func monitorV2LinkColumnMetaDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2LinkColumnMetaInput
		Schema: map[string]*schema.Schema{
			"src_fields": { // [MonitorV2ColumnPathInput!]
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        monitorV2ColumnPathResource(),
				Description: descriptions.Get("monitorv2", "schema", "link_column_meta", "src_fields"),
			},
			"dst_fields": { // [String!]
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: descriptions.Get("monitorv2", "schema", "link_column_meta", "dst_fields"),
			},
			"target_dataset": { // Int64
				Type:        schema.TypeInt,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "link_column_meta", "target_dataset"),
			},
		},
	}
}

func monitorV2LinkColumnDatasource() *schema.Resource {
	return &schema.Resource{ // MonitorV2LinkColumnInput
		Schema: map[string]*schema.Schema{
			"name": { // String!
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "link_column", "name"),
			},
			"meta": { // MonitorV2LinkColumnMetaInput
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        monitorV2LinkColumnMetaResource(),
				Description: descriptions.Get("monitorv2", "schema", "link_column_meta", "description"),
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
				MaxItems:    1,
				Elem:        monitorV2LinkColumnResource(),
				Description: descriptions.Get("monitorv2", "schema", "link_column", "description"),
			},
			"column_path": { // MonitorV2ColumnPathInput
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        monitorV2ColumnPathResource(),
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
				Required:    true,
				MinItems:    1,
				Elem:        monitorV2ComparisonResource(),
				Description: descriptions.Get("monitorv2", "schema", "compare_values"),
			},
			"column": { // MonitorV2ColumnInput!
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Elem:        monitorV2ColumnResource(),
				Description: descriptions.Get("monitorv2", "schema", "column", "description"),
			},
		},
	}
}

func dataSourceMonitorV2Read(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	monitor, err := client.GetMonitorV2(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitorv2: %s", err.Error())
	}

	// perform data.set on all the fields from this monitor
	if err := data.Set("workspace_id", oid.WorkspaceOid(monitor.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", monitor.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", monitor.IconUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitor.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("id", monitor.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("rule_kind", toSnake(string(monitor.GetRuleKind()))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	_, err = flattenAndSetQuery(data, monitor.Definition.InputQuery.Stages, monitor.Definition.InputQuery.OutputStage)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("rules", monitorV2FlattenRules(monitor.Definition.Rules)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("lookback_time", monitor.Definition.LookbackTime.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if monitor.Definition.DataStabilizationDelay != nil {
		if err := data.Set("data_stabilization_delay", monitor.Definition.DataStabilizationDelay.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if monitor.Definition.Groupings != nil {
		if err := data.Set("groupings", monitorV2FlattenGroupings(monitor.Definition.Groupings)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if monitor.Definition.Scheduling != nil {
		if err := data.Set("scheduling", monitorV2FlattenScheduling(*monitor.Definition.Scheduling)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}
