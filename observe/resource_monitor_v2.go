package observe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("monitorv2", "description"),
		CreateContext: resourceMonitorV2Create,
		ReadContext:   resourceMonitorV2Read,
		UpdateContext: resourceMonitorV2Update,
		DeleteContext: resourceMonitorV2Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// needed as input to MonitorV2Create, also part of MonitorV2 struct
			"workspace": { // ObjectId!
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("monitorv2", "schema", "workspace_id"),
			},
			// fields of MonitorV2Input excluding the components of MonitorV2DefinitionInput
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
				Optional:    true,
				MaxItems:    1,
				Description: descriptions.Get("monitorv2", "schema", "scheduling", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"transform": { // MonitorV2TransformScheduleInput
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: descriptions.Get("monitorv2", "schema", "scheduling", "transform", "description"),
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
			// the following field describes how monitorv2 is connected to shared actions.
			"actions": { // [MonitorV2ActionRuleInput]
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"oid": { // ObjectId!
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateOID(oid.TypeMonitorV2Action),
							Description:      descriptions.Get("monitorv2", "schema", "actions", "oid"),
						},
						"levels": { // [MonitorV2AlarmLevel!]
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateEnums(gql.AllMonitorV2AlarmLevels),
							},
							Description: descriptions.Get("monitorv2", "schema", "actions", "levels"),
						},
						"send_end_notifications": { // Boolean
							Type:        schema.TypeBool,
							Optional:    true,
							Description: descriptions.Get("monitorv2", "schema", "actions", "send_end_notifications"),
						},
						"send_reminders_interval": { // Duration
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validateTimeDuration,
							DiffSuppressFunc: diffSuppressTimeDuration,
							Description:      descriptions.Get("monitorv2", "schema", "actions", "send_reminders_interval"),
						},
					},
				},
				Description: descriptions.Get("monitorv2", "schema", "actions", "description"),
			},
			// the following fields are those that aren't given as input to CU ops, but can be read by R ops.
			"oid": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2ComparisonResource() *schema.Resource {
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
					Type:             schema.TypeInt,
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

func monitorV2ColumnPathResource() *schema.Resource {
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

func monitorV2LinkColumnResource() *schema.Resource {
	return &schema.Resource{ // MonitorV2LinkColumnInput
		Schema: map[string]*schema.Schema{
			"name": { // String!
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "link_column", "name"),
			},
		},
	}
}

func monitorV2ColumnResource() *schema.Resource {
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

func monitorV2ColumnComparisonResource() *schema.Resource {
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

func resourceMonitorV2Create(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2Input(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitorV2(ctx, id.Id, input)
	if err != nil {
		return diag.Errorf("failed to create monitor: %s", err.Error())
	}

	result, err = relateMonitorV2ToActions(ctx, result.Id, data, client)
	if err != nil {
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceMonitorV2Read(ctx, data, meta)...)
}

func resourceMonitorV2Update(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2Input(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2(ctx, data.Id(), input)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2Create(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to update monitor: %s", err.Error())
	}

	_, err = relateMonitorV2ToActions(ctx, data.Id(), data, client)
	if err != nil {
		return diag.Errorf("failed to update monitor: %s", err.Error())
	}

	return append(diags, resourceMonitorV2Read(ctx, data, meta)...)
}

func resourceMonitorV2Read(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
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
	if err := data.Set("workspace", oid.WorkspaceOid(monitor.WorkspaceId).String()); err != nil {
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

	if err := data.Set("oid", monitor.Oid().String()); err != nil {
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

	if len(monitor.ActionRules) > 0 {
		if err := data.Set("actions", monitorV2FlattenActionRules(monitor.ActionRules)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func resourceMonitorV2Delete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorV2(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor: %s", err.Error())
	}
	return diags
}

func monitorV2FlattenRules(gqlRules []gql.MonitorV2Rule) []interface{} {
	var rules []interface{}
	for _, gqlRule := range gqlRules {
		rules = append(rules, monitorV2FlattenRule(gqlRule))
	}
	return rules
}

func monitorV2FlattenRule(gqlRule gql.MonitorV2Rule) interface{} {
	rule := map[string]interface{}{
		"level": toSnake(string(gqlRule.Level)),
	}
	if gqlRule.Count != nil {
		rule["count"] = monitorV2FlattenCountRule(*gqlRule.Count)
	}
	if gqlRule.Threshold != nil {
		rule["threshold"] = monitorV2FlattenThresholdRule(*gqlRule.Threshold)
	}
	if gqlRule.Promote != nil {
		rule["promote"] = monitorV2FlattenPromoteRule(*gqlRule.Promote)
	}
	return rule
}

func monitorV2FlattenActionRules(gqlActionRules []gql.MonitorV2ActionRule) []interface{} {
	var actionRules []interface{}
	for _, gqlActionRule := range gqlActionRules {
		actionRules = append(actionRules, monitorV2FlattenActionRule(gqlActionRule))
	}
	return actionRules
}

func monitorV2FlattenActionRule(gqlActionRule gql.MonitorV2ActionRule) interface{} {
	rules := map[string]interface{}{
		"oid": oid.MonitorV2ActionOid(gqlActionRule.ActionID).String(),
	}
	if len(gqlActionRule.Levels) > 0 {
		levels := make([]interface{}, 0)
		for _, level := range gqlActionRule.Levels {
			levels = append(levels, toSnake(string(level)))
		}
		rules["levels"] = levels
	}
	if gqlActionRule.SendEndNotifications != nil {
		rules["send_end_notifications"] = *gqlActionRule.SendEndNotifications
	}
	if gqlActionRule.SendRemindersInterval != nil {
		rules["send_reminders_interval"] = gqlActionRule.SendRemindersInterval.String()
	}
	return rules
}

func monitorV2FlattenCountRule(gqlCount gql.MonitorV2CountRule) []interface{} {
	countRule := map[string]interface{}{}
	if gqlCount.CompareValues != nil {
		countRule["compare_values"] = monitorV2FlattenComparisons(gqlCount.CompareValues)
	}
	if gqlCount.CompareGroups != nil {
		countRule["compare_groups"] = monitorV2FlattenColumnComparisons(gqlCount.CompareGroups)
	}
	return []interface{}{countRule}
}

func monitorV2FlattenThresholdRule(gqlThreshold gql.MonitorV2ThresholdRule) []interface{} {
	thresholdRule := map[string]interface{}{
		"value_column_name": gqlThreshold.ValueColumnName,
		"aggregation":       toSnake(string(gqlThreshold.Aggregation)),
	}
	if gqlThreshold.CompareValues != nil {
		thresholdRule["compare_values"] = monitorV2FlattenComparisons(gqlThreshold.CompareValues)
	}
	if gqlThreshold.CompareGroups != nil {
		thresholdRule["compare_groups"] = monitorV2FlattenColumnComparisons(gqlThreshold.CompareGroups)
	}
	return []interface{}{thresholdRule}
}

func monitorV2FlattenPromoteRule(gqlPromote gql.MonitorV2PromoteRule) []interface{} {
	promoteRule := map[string]interface{}{}
	if gqlPromote.CompareColumns != nil {
		promoteRule["compare_columns"] = monitorV2FlattenColumnComparisons(gqlPromote.CompareColumns)
	}
	return []interface{}{promoteRule}
}

func monitorV2FlattenColumnComparisons(gqlColumnComparisons []gql.MonitorV2ColumnComparison) []interface{} {
	columnComparisons := []interface{}{}
	for _, gqlColumnComparison := range gqlColumnComparisons {
		columnComparisons = append(columnComparisons, monitorV2FlattenColumnComparison(gqlColumnComparison))
	}
	return columnComparisons
}

func monitorV2FlattenColumnComparison(gqlColumnComparison gql.MonitorV2ColumnComparison) interface{} {
	columnComparison := map[string]interface{}{
		"column": monitorV2FlattenColumn(gqlColumnComparison.Column),
	}
	if gqlColumnComparison.CompareValues != nil {
		columnComparison["compare_values"] = monitorV2FlattenComparisons(gqlColumnComparison.CompareValues)
	}
	return columnComparison
}

func monitorV2FlattenColumn(gqlColumn gql.MonitorV2Column) []interface{} {
	column := map[string]interface{}{}
	if gqlColumn.LinkColumn != nil {
		column["link_column"] = monitorV2FlattenLinkColumn(*gqlColumn.LinkColumn)
	}
	if gqlColumn.ColumnPath != nil {
		column["column_path"] = monitorV2FlattenColumnPath(*gqlColumn.ColumnPath)
	}
	return []interface{}{column}
}

func monitorV2FlattenComparisons(gqlComparisons []gql.MonitorV2Comparison) []interface{} {
	comparisons := []interface{}{}
	for _, gqlComparison := range gqlComparisons {
		comparisons = append(comparisons, monitorV2FlattenComparison(gqlComparison))
	}
	return comparisons
}

func monitorV2FlattenComparison(gqlComparison gql.MonitorV2Comparison) interface{} {
	comparison := map[string]interface{}{
		"compare_fn": toSnake(string(gqlComparison.CompareFn)),
	}
	monitorV2FlattenPrimitiveValue(gqlComparison.CompareValue, comparison)
	return comparison
}

func monitorV2FlattenPrimitiveValue(gqlPrimitiveValue gql.PrimitiveValue, primitiveValue map[string]interface{}) {
	if gqlPrimitiveValue.Bool != nil {
		primitiveValue["value_bool"] = []interface{}{*gqlPrimitiveValue.Bool}
	}
	if gqlPrimitiveValue.Int64 != nil {
		primitiveValue["value_int64"] = []interface{}{int(*gqlPrimitiveValue.Int64)}
	}
	if gqlPrimitiveValue.Float64 != nil {
		primitiveValue["value_float64"] = []interface{}{*gqlPrimitiveValue.Float64}
	}
	if gqlPrimitiveValue.String != nil {
		primitiveValue["value_string"] = []interface{}{*gqlPrimitiveValue.String}
	}
	if gqlPrimitiveValue.Timestamp != nil {
		primitiveValue["value_timestamp"] = []interface{}{gqlPrimitiveValue.Timestamp.String()}
	}
	if gqlPrimitiveValue.Duration != nil {
		primitiveValue["value_duration"] = []interface{}{gqlPrimitiveValue.Duration.String()}
	}
}

func monitorV2FlattenGroupings(gqlGroupings []gql.MonitorV2Column) []interface{} {
	var groupings []interface{}
	for _, gqlGrouping := range gqlGroupings {
		grouping := map[string]interface{}{}
		if gqlGrouping.LinkColumn != nil {
			grouping["link_column"] = monitorV2FlattenLinkColumn(*gqlGrouping.LinkColumn)
		}
		if gqlGrouping.ColumnPath != nil {
			grouping["column_path"] = monitorV2FlattenColumnPath(*gqlGrouping.ColumnPath)
		}
		groupings = append(groupings, grouping)
	}
	return groupings
}

func monitorV2FlattenColumnPaths(gqlColumnPaths []gql.MonitorV2ColumnPath) []interface{} {
	var columnPaths []interface{}
	for _, gqlColumnPath := range gqlColumnPaths {
		columnPaths = append(columnPaths, monitorV2FlattenColumnPath(gqlColumnPath))
	}
	return columnPaths
}

func monitorV2FlattenColumnPath(gqlColumnPath gql.MonitorV2ColumnPath) []interface{} {
	columnPath := map[string]interface{}{
		"name": gqlColumnPath.Name,
	}
	if gqlColumnPath.Path != nil {
		columnPath["path"] = *gqlColumnPath.Path
	}
	return []interface{}{columnPath}
}

func monitorV2FlattenLinkColumn(gqlLinkColumn gql.MonitorV2LinkColumn) []interface{} {
	linkColumn := map[string]interface{}{
		"name": gqlLinkColumn.Name,
	}
	return []interface{}{linkColumn}
}

func monitorV2FlattenScheduling(gqlScheduling gql.MonitorV2Scheduling) []interface{} {
	scheduling := map[string]interface{}{}
	if gqlScheduling.Interval != nil {
		scheduling["interval"] = monitorV2FlattenIntervalSchedule(*gqlScheduling.Interval)
	}
	if gqlScheduling.Transform != nil {
		scheduling["transform"] = monitorV2FlattenTransformSchedule(*gqlScheduling.Transform)
	}
	return []interface{}{scheduling}
}

func monitorV2FlattenIntervalSchedule(gqlIntervalSchedule gql.MonitorV2IntervalSchedule) []interface{} {
	intervalSchedule := map[string]interface{}{
		"interval":  gqlIntervalSchedule.Interval.String(),
		"randomize": gqlIntervalSchedule.Randomize.String(),
	}
	return []interface{}{intervalSchedule}
}

func monitorV2FlattenTransformSchedule(gqlTransformSchedule gql.MonitorV2TransformSchedule) []interface{} {
	transformSchedule := map[string]interface{}{
		"freshness_goal": gqlTransformSchedule.FreshnessGoal.String(),
	}
	return []interface{}{transformSchedule}
}

func newMonitorV2Input(data *schema.ResourceData) (input *gql.MonitorV2Input, diags diag.Diagnostics) {
	// required
	definitionInput, diags := newMonitorV2DefinitionInput(data)
	if diags.HasError() {
		return nil, diags
	}
	ruleKind := toCamel(data.Get("rule_kind").(string))
	name := data.Get("name").(string)

	// instantiation
	input = &gql.MonitorV2Input{
		Definition: *definitionInput,
		RuleKind:   meta.MonitorV2RuleKind(ruleKind),
		Name:       name,
	}

	// optionals
	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return input, diags
}

func newMonitorV2DefinitionInput(data *schema.ResourceData) (defnInput *gql.MonitorV2DefinitionInput, diags diag.Diagnostics) {
	// required
	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, diags
	}
	if query == nil {
		return nil, diag.Errorf("no query provided")
	}
	rules := make([]gql.MonitorV2RuleInput, 0)
	for i := range data.Get("rules").([]interface{}) {
		rule, diags := newMonitorV2RuleInput(fmt.Sprintf("rules.%d.", i), data)
		if diags.HasError() {
			return nil, diags
		}
		rules = append(rules, *rule)
	}
	scheduling, diags := newMonitorV2SchedulingInput("scheduling.0.", data)
	if diags.HasError() {
		return nil, diags
	}

	// instantiation
	defnInput = &gql.MonitorV2DefinitionInput{
		InputQuery: *query,
		Rules:      rules,
		Scheduling: scheduling,
	}

	// optionals
	if v, ok := data.GetOk("data_stabilization_delay"); ok {
		dataStabilizationDelay, _ := types.ParseDurationScalar(v.(string))
		defnInput.DataStabilizationDelay = dataStabilizationDelay
	}
	if _, ok := data.GetOk("groupings"); ok {
		groupings := make([]gql.MonitorV2ColumnInput, 0)
		for i := range data.Get("groupings").([]interface{}) {
			colInput, diags := newMonitorV2ColumnInput(fmt.Sprintf("groupings.%d.", i), data)
			if diags.HasError() {
				return nil, diags
			}
			groupings = append(groupings, *colInput)
		}
		defnInput.Groupings = groupings
	}
	if lookbackTimeStr, ok := data.GetOk("lookback_time"); ok {
		lookbackTime, err := types.ParseDurationScalar(lookbackTimeStr.(string))
		if err != nil {
			return nil, diag.Errorf("lookback_time is invalid: %s", err.Error())
		}
		defnInput.LookbackTime = lookbackTime
	} else {
		lookbackTime, err := types.ParseDurationScalar("0")
		if err != nil {
			return nil, diag.Errorf("lookback_time is invalid: %s", err.Error())
		}
		defnInput.LookbackTime = lookbackTime
	}

	return defnInput, diags
}

func newMonitorV2SchedulingInput(path string, data *schema.ResourceData) (scheduling *gql.MonitorV2SchedulingInput, diags diag.Diagnostics) {
	// instantiation
	scheduling = &gql.MonitorV2SchedulingInput{}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%sinterval", path)); ok {
		interval, diags := newMonitorV2IntervalScheduleInput(fmt.Sprintf("%sinterval.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		scheduling.Interval = interval
	}
	if _, ok := data.GetOk(fmt.Sprintf("%stransform", path)); ok {
		transform, diags := newMonitorV2TransformScheduleInput(fmt.Sprintf("%stransform.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		scheduling.Transform = transform
	}

	if scheduling.Interval == nil && scheduling.Transform == nil {
		return nil, diags
	}

	return scheduling, diags
}

func newMonitorV2IntervalScheduleInput(path string, data *schema.ResourceData) (interval *gql.MonitorV2IntervalScheduleInput, diags diag.Diagnostics) {
	// required
	intervalField := data.Get(fmt.Sprintf("%sinterval", path)).(string)
	randomizeField := data.Get(fmt.Sprintf("%srandomize", path)).(string)
	intervalDuration, _ := types.ParseDurationScalar(intervalField)
	randomizeDuration, _ := types.ParseDurationScalar(randomizeField)

	// instantiation
	interval = &gql.MonitorV2IntervalScheduleInput{
		Interval:  *intervalDuration,
		Randomize: *randomizeDuration,
	}

	return interval, diags
}

func newMonitorV2TransformScheduleInput(path string, data *schema.ResourceData) (transform *gql.MonitorV2TransformScheduleInput, diags diag.Diagnostics) {
	// required
	transformField := data.Get(fmt.Sprintf("%sfreshness_goal", path)).(string)
	transformDuration, _ := types.ParseDurationScalar(transformField)

	// instantiation
	transform = &gql.MonitorV2TransformScheduleInput{FreshnessGoal: *transformDuration}

	return transform, diags
}

func newMonitorV2RuleInput(path string, data *schema.ResourceData) (rule *gql.MonitorV2RuleInput, diags diag.Diagnostics) {
	// required
	level := toCamel(data.Get(fmt.Sprintf("%slevel", path)).(string))

	// instantiation
	rule = &gql.MonitorV2RuleInput{Level: gql.MonitorV2AlarmLevel(level)}

	// optionals
	nRules := 0
	if _, ok := data.GetOk(fmt.Sprintf("%scount", path)); ok {
		count, diags := newMonitorV2CountRuleInput(fmt.Sprintf("%scount.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		rule.Count = count
		nRules++
	}
	if _, ok := data.GetOk(fmt.Sprintf("%sthreshold", path)); ok {
		threshold, diags := newMonitorV2ThresholdRuleInput(fmt.Sprintf("%sthreshold.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		rule.Threshold = threshold
		nRules++
	}
	if _, ok := data.GetOk(fmt.Sprintf("%spromote", path)); ok {
		promote, diags := newMonitorV2PromoteRuleInput(fmt.Sprintf("%spromote.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		rule.Promote = promote
		nRules++
	}
	if nRules != 1 {
		return nil, diag.Errorf("exactly one of count, threshold, or promote must be specified")
	}

	return rule, diags
}

func newMonitorV2CountRuleInput(path string, data *schema.ResourceData) (comparison *gql.MonitorV2CountRuleInput, diags diag.Diagnostics) {
	// required
	comparisonInputs := make([]gql.MonitorV2ComparisonInput, 0)
	for i := range data.Get(fmt.Sprintf("%scompare_values", path)).([]interface{}) {
		comparisonInput, diags := newMonitorV2ComparisonInput(fmt.Sprintf("%scompare_values.%d.", path, i), data)
		if diags.HasError() {
			return nil, diags
		}
		comparisonInputs = append(comparisonInputs, *comparisonInput)
	}

	// instantiation
	comparison = &gql.MonitorV2CountRuleInput{
		CompareValues: comparisonInputs,
	}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%scompare_groups", path)); ok {
		compareGroups := make([]gql.MonitorV2ColumnComparisonInput, 0)
		for i := range data.Get(fmt.Sprintf("%scompare_groups", path)).([]interface{}) {
			columnComparison, diags := newMonitorV2ColumnComparisonInput(fmt.Sprintf("%scompare_groups.%d.", path, i), data)
			if diags.HasError() {
				return nil, diags
			}
			compareGroups = append(compareGroups, *columnComparison)
		}
		comparison.CompareGroups = compareGroups
	}

	return comparison, diags
}

func newMonitorV2ComparisonInput(path string, data *schema.ResourceData) (comparison *gql.MonitorV2ComparisonInput, diags diag.Diagnostics) {
	// required
	compareFn := gql.MonitorV2ComparisonFunction(toCamel(data.Get(fmt.Sprintf("%scompare_fn", path)).(string)))
	var compareValue gql.PrimitiveValueInput
	diags = newMonitorV2PrimitiveValue(path, data, &compareValue)
	if diags.HasError() {
		return nil, diags
	}

	// instantiation
	comparison = &gql.MonitorV2ComparisonInput{
		CompareFn:    compareFn,
		CompareValue: compareValue,
	}

	return comparison, diags
}

func newMonitorV2ThresholdRuleInput(path string, data *schema.ResourceData) (threshold *gql.MonitorV2ThresholdRuleInput, diags diag.Diagnostics) {
	// required
	compareValues := []gql.MonitorV2ComparisonInput{}
	for i := range data.Get(fmt.Sprintf("%scompare_values", path)).([]interface{}) {
		comparisonInput, diags := newMonitorV2ComparisonInput(fmt.Sprintf("%scompare_values.%d.", path, i), data)
		if diags.HasError() {
			return threshold, diags
		}
		compareValues = append(compareValues, *comparisonInput)
	}

	valueColumnName := data.Get(fmt.Sprintf("%svalue_column_name", path)).(string)
	aggregation := gql.MonitorV2ValueAggregation(toCamel(data.Get(fmt.Sprintf("%saggregation", path)).(string)))

	// instantiation
	threshold = &gql.MonitorV2ThresholdRuleInput{
		CompareValues:   compareValues,
		ValueColumnName: valueColumnName,
		Aggregation:     aggregation,
	}

	return threshold, diags
}

func newMonitorV2PromoteRuleInput(prefix string, data *schema.ResourceData) (promoteRule *gql.MonitorV2PromoteRuleInput, diags diag.Diagnostics) {
	// instantiation
	promoteRule = &gql.MonitorV2PromoteRuleInput{}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%scompare_columns", prefix)); ok {
		compareColumns := make([]gql.MonitorV2ColumnComparisonInput, 0)
		for i := range data.Get(fmt.Sprintf("%scompare_columns", prefix)).([]interface{}) {
			input, diags := newMonitorV2ColumnComparisonInput(fmt.Sprintf("%scompare_columns.%d.", prefix, i), data)
			if diags.HasError() {
				return nil, diags
			}
			compareColumns = append(compareColumns, *input)
		}
		promoteRule.CompareColumns = compareColumns
	}

	return promoteRule, diags
}

func newMonitorV2ColumnComparisonInput(path string, data *schema.ResourceData) (comparison *gql.MonitorV2ColumnComparisonInput, diags diag.Diagnostics) {
	// required
	compareValues := make([]gql.MonitorV2ComparisonInput, 0)
	for i := range data.Get(fmt.Sprintf("%scompare_values", path)).([]interface{}) {
		comparisonInput, diags := newMonitorV2ComparisonInput(fmt.Sprintf("%scompare_values.%d.", path, i), data)
		if diags.HasError() {
			return nil, diags
		}
		compareValues = append(compareValues, *comparisonInput)
	}
	columnInput, diags := newMonitorV2ColumnInput(fmt.Sprintf("%scolumn.0.", path), data)
	if diags.HasError() {
		return nil, diags
	}

	// instantiation
	comparison = &gql.MonitorV2ColumnComparisonInput{
		Column:        *columnInput,
		CompareValues: compareValues,
	}

	return comparison, diags
}

func newMonitorV2ColumnInput(path string, data *schema.ResourceData) (column *gql.MonitorV2ColumnInput, diags diag.Diagnostics) {
	// instantiation
	column = &gql.MonitorV2ColumnInput{}

	// optional
	if _, ok := data.GetOk(fmt.Sprintf("%slink_column", path)); ok {
		linkColumn, diags := newMonitorV2LinkColumnInput(fmt.Sprintf("%slink_column.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		column.LinkColumn = linkColumn
	}

	if _, ok := data.GetOk(fmt.Sprintf("%scolumn_path", path)); ok {
		columnPath, diags := newMonitorV2ColumnPathInput(fmt.Sprintf("%scolumn_path.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		column.ColumnPath = columnPath
	}

	return column, diags
}

func newMonitorV2LinkColumnInput(path string, data *schema.ResourceData) (column *gql.MonitorV2LinkColumnInput, diags diag.Diagnostics) {
	// required
	name := data.Get(fmt.Sprintf("%sname", path)).(string)

	// instantiation
	column = &gql.MonitorV2LinkColumnInput{Name: name}

	return column, diags
}

func newMonitorV2ColumnPathInput(path string, data *schema.ResourceData) (column *gql.MonitorV2ColumnPathInput, diags diag.Diagnostics) {
	// required
	name := data.Get(fmt.Sprintf("%sname", path)).(string)

	// instantiation
	column = &gql.MonitorV2ColumnPathInput{Name: name}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%spath", path)); ok {
		p := v.(string)
		column.Path = &p
	}

	return column, diags
}

func newMonitorV2PrimitiveValue(path string, data *schema.ResourceData, ret *gql.PrimitiveValueInput) diag.Diagnostics {
	valueBool, hasBool := data.GetOk(fmt.Sprintf("%svalue_bool", path))
	valueInt, hasInt := data.GetOk(fmt.Sprintf("%svalue_int64", path))
	valueFloat, hasFloat := data.GetOk(fmt.Sprintf("%svalue_float64", path))
	valueString, hasString := data.GetOk(fmt.Sprintf("%svalue_string", path))
	valueDuration, hasDuration := data.GetOk(fmt.Sprintf("%svalue_duration", path))
	valueTimestamp, hasTimestamp := data.GetOk(fmt.Sprintf("%svalue_timestamp", path))

	nvalue := 0
	var kinds []string
	if hasBool && valueBool != nil {
		b := valueBool.([]interface{})[0].(bool)
		ret.Bool = &b
		nvalue++
		kinds = append(kinds, "value_bool")
	}
	if hasInt && valueInt != nil {
		i64 := types.Int64Scalar(valueInt.([]interface{})[0].(int))
		ret.Int64 = &i64
		nvalue++
		kinds = append(kinds, "value_int64")
	}
	if hasFloat && valueFloat != nil {
		vlt := valueFloat.([]interface{})[0].(float64)
		ret.Float64 = &vlt
		nvalue++
		kinds = append(kinds, "value_float64")
	}
	if hasString && valueString != nil {
		vstr := valueString.([]interface{})[0].(string)
		ret.String = &vstr
		nvalue++
		kinds = append(kinds, "value_string")
	}
	if hasDuration && valueDuration != nil {
		dur, _ := time.ParseDuration(valueDuration.([]interface{})[0].(string))
		i64 := types.Int64Scalar(dur.Nanoseconds())
		ret.Duration = &i64
		nvalue++
		kinds = append(kinds, "value_duration")
	}
	if hasTimestamp && valueTimestamp != nil {
		tsp, _ := time.Parse(time.RFC3339, valueTimestamp.([]interface{})[0].(string))
		tss := types.TimeScalar(tsp)
		ret.Timestamp = &tss
		nvalue++
		kinds = append(kinds, "value_timestamp")
	}
	if nvalue == 0 {
		return diag.Errorf("A value must be specified (value_string, value_bool, etc). Path = %s", path)
	}
	if nvalue > 1 {
		return diag.Errorf("Only one value may be specified (value_string, value_bool, etc); there are %d: %s. Path = %s", len(kinds), strings.Join(kinds, ","), path)
	}
	return nil
}

func relateMonitorV2ToActions(ctx context.Context, monitorId string, data *schema.ResourceData, client *observe.Client) (*gql.MonitorV2, error) {
	var actionRelations []gql.ActionRelationInput
	if _, ok := data.GetOk("actions"); ok {
		actionRelations = make([]gql.ActionRelationInput, 0)
		for i := range data.Get("actions").([]interface{}) {
			actionRule, err := newMonitorV2ActionRuleInput(fmt.Sprintf("actions.%d.", i), data)
			if err != nil {
				return nil, err
			}
			actionRelations = append(actionRelations, gql.ActionRelationInput{
				ActionRule: *actionRule,
			})
		}
	}
	return client.SaveMonitorV2Relations(ctx, monitorId, actionRelations)
}

func newMonitorV2ActionRuleInput(path string, data *schema.ResourceData) (*gql.MonitorV2ActionRuleInput, error) {
	// required
	actOID, err := oid.NewOID(data.Get(fmt.Sprintf("%soid", path)).(string))
	if err != nil {
		return nil, err
	}

	// instantiation
	act := &gql.MonitorV2ActionRuleInput{
		ActionID: actOID.Id,
	}

	// optional
	if _, ok := data.GetOk(fmt.Sprintf("%slevels", path)); ok {
		act.Levels = make([]gql.MonitorV2AlarmLevel, 0)
		for i := range data.Get(fmt.Sprintf("%slevels", path)).([]interface{}) {
			act.Levels = append(act.Levels, gql.MonitorV2AlarmLevel(toCamel(data.Get(fmt.Sprintf("%slevels.%d", path, i)).(string))))
		}
	}

	if v, ok := data.GetOk(fmt.Sprintf("%ssend_end_notifications", path)); ok {
		boolVal := v.(bool)
		act.SendEndNotifications = &boolVal
	}

	if v, ok := data.GetOk(fmt.Sprintf("%ssend_reminders_interval", path)); ok {
		stringVal := v.(string)
		interval, _ := types.ParseDurationScalar(stringVal)
		act.SendRemindersInterval = interval
	}

	return act, nil
}
