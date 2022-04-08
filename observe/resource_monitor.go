package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaMonitorWorkspaceDescription   = "OID of workspace monitor is contained in."
	schemaMonitorNameDescription        = "Monitor name."
	schemaMonitorDescriptionDescription = "Monitor description."
	schemaMonitorIconDescription        = "Icon image."
	schemaMonitorOIDDescription         = "The Observe ID for monitor."
)

var validRules = []string{
	"rule.0.change",
	"rule.0.count",
	"rule.0.facet",
	"rule.0.threshold",
	"rule.0.promote",
}

func resourceMonitor() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorCreate,
		ReadContext:   resourceMonitorRead,
		UpdateContext: resourceMonitorUpdate,
		DeleteContext: resourceMonitorDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"freshness": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
			},
			"disabled": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"stage": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
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
						},
						"input": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"pipeline": {
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: diffSuppressPipeline,
						},
					},
				},
			},
			"rule": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_column": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"group_by_group": {
							Type:          schema.TypeList,
							Optional:      true,
							ConflictsWith: []string{"rule.0.group_by"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"columns": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"group_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"group_by": {
							Type:             schema.TypeString,
							Deprecated:       "Use \"group_by_group\" instead",
							Optional:         true,
							Default:          "none",
							ValidateDiagFunc: validateEnums(observe.MonitorGroupings),
						},
						"count": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.CompareFunctions),
									},
									"compare_value": {
										Type:     schema.TypeFloat,
										Optional: true,
										// relax restriction during phase out
										// ExactlyOneOf: []string{"rule.0.count.0.compare_value", "rule.0.count.0.compare_values"},
										Deprecated: "Use compare_values instead",
									},
									"compare_values": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 2,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:             schema.TypeString,
										Required:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
								},
							},
						},
						"change": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"change_type": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "absolute",
										ValidateDiagFunc: validateEnums(observe.ChangeTypes),
									},
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.CompareFunctions),
									},
									"aggregate_function": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "avg",
										ValidateDiagFunc: validateEnums(observe.AggregateFunctions),
									},
									"compare_value": {
										Type:     schema.TypeFloat,
										Optional: true,
										// relax restriction during phase out
										//ExactlyOneOf: []string{"rule.0.change.0.compare_value", "rule.0.change.0.compare_values"},
										Deprecated: "Use compare_values instead",
									},
									"compare_values": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 2,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:             schema.TypeString,
										Required:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
									"baseline_time": {
										Type:             schema.TypeString,
										Required:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
								},
							},
						},
						"facet": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"facet_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.FacetFunctions),
									},
									"facet_values": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"time_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.TimeFunctions),
									},
									"time_value": {
										Type:     schema.TypeFloat,
										Optional: true,
									},
									"lookback_time": {
										Type:             schema.TypeString,
										Required:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
								},
							},
						},
						"threshold": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.CompareFunctions),
									},
									"compare_values": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 2,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:             schema.TypeString,
										Required:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
								},
							},
						},
						"promote": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"primary_key": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"kind_field": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"description_field": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"notification_spec": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"importance": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "informational",
							ValidateDiagFunc: validateEnums(observe.NotificationImportances),
						},
						"merge": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "merged",
							ValidateDiagFunc: validateEnums(observe.NotificationMerges),
						},
					},
				},
			},
		},
	}
}

func newMonitorRuleConfig(data *schema.ResourceData) (ruleConfig *observe.MonitorRuleConfig, diags diag.Diagnostics) {
	ruleConfig = &observe.MonitorRuleConfig{}

	if v, ok := data.GetOk("rule.0.source_column"); ok {
		s := v.(string)
		ruleConfig.SourceColumn = &s
	}

	if v, ok := data.GetOk("rule.0.group_by"); ok {
		g := observe.MonitorGrouping(toCamel(v.(string)))
		ruleConfig.GroupBy = &g
	}

	if v, ok := data.GetOk("rule.0.group_by_group"); ok {
		for _, el := range v.([]interface{}) {
			info := observe.MonitorGroupInfo{
				Columns: make([]string, 0),
			}
			if el != nil {
				value := el.(map[string]interface{})
				info.GroupName = value["group_name"].(string)
				for _, col := range value["columns"].([]interface{}) {
					info.Columns = append(info.Columns, col.(string))
				}
			}
			ruleConfig.GroupByGroups = append(ruleConfig.GroupByGroups, info)
		}
	}

	if data.Get("rule.0.count.#") == 1 {
		ruleConfig.CountRule = &observe.MonitorRuleCountConfig{}

		v := data.Get("rule.0.count.0.compare_function")
		fn := observe.CompareFunction(toCamel(v.(string)))
		ruleConfig.CountRule.CompareFunction = &fn

		// TODO: remove compare_value
		if v, ok := data.GetOk("rule.0.count.0.compare_value"); ok {
			ruleConfig.CountRule.CompareValues = []float64{v.(float64)}
		} else if v, ok := data.GetOk("rule.0.count.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				ruleConfig.CountRule.CompareValues = append(ruleConfig.CountRule.CompareValues, i.(float64))
			}
		}
		if v, ok := data.GetOk("rule.0.count.0.lookback_time"); ok {
			t, _ := time.ParseDuration(v.(string))
			ruleConfig.CountRule.LookbackTime = &t
		}
	}

	if data.Get("rule.0.change.#") == 1 {
		ruleConfig.ChangeRule = &observe.MonitorRuleChangeConfig{}

		if v := data.Get("rule.0.change.0.change_type"); true {
			ruleConfig.ChangeRule.ChangeType = observe.ChangeType(toCamel(v.(string)))
		}

		if v := data.Get("rule.0.change.0.aggregate_function"); true {
			fn := observe.AggregateFunction(toCamel(v.(string)))
			ruleConfig.ChangeRule.AggregateFunction = &fn
		}

		if v := data.Get("rule.0.change.0.compare_function"); true {
			fn := observe.CompareFunction(toCamel(v.(string)))
			ruleConfig.ChangeRule.CompareFunction = &fn
		}

		// TODO: remove compare_value
		if v, ok := data.GetOk("rule.0.change.0.compare_value"); ok {
			ruleConfig.ChangeRule.CompareValues = []float64{v.(float64)}
		} else if v, ok := data.GetOk("rule.0.change.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				ruleConfig.ChangeRule.CompareValues = append(ruleConfig.ChangeRule.CompareValues, i.(float64))
			}
		}

		if v, ok := data.GetOk("rule.0.change.0.lookback_time"); ok {
			t, _ := time.ParseDuration(v.(string))
			ruleConfig.ChangeRule.LookbackTime = &t
		}

		if v, ok := data.GetOk("rule.0.change.0.baseline_time"); ok {
			t, _ := time.ParseDuration(v.(string))
			ruleConfig.ChangeRule.BaselineTime = &t
		}
	}

	if data.Get("rule.0.facet.#") == 1 {
		ruleConfig.FacetRule = &observe.MonitorRuleFacetConfig{}

		if v, ok := data.GetOk("rule.0.facet.0.facet_function"); ok {
			fn := observe.FacetFunction(toCamel(v.(string)))
			ruleConfig.FacetRule.FacetFunction = &fn
		}

		if v, ok := data.GetOk("rule.0.facet.0.facet_values"); ok {
			var values []string
			for _, el := range v.([]interface{}) {
				values = append(values, el.(string))
			}
			ruleConfig.FacetRule.FacetValues = values
		}

		if v, ok := data.GetOk("rule.0.facet.0.time_function"); ok {
			fn := observe.TimeFunction(toCamel(v.(string)))
			ruleConfig.FacetRule.TimeFunction = &fn
		}

		if v, ok := data.GetOk("rule.0.facet.0.time_value"); ok {
			f := v.(float64)
			ruleConfig.FacetRule.TimeValue = &f
		}

		if v, ok := data.GetOk("rule.0.facet.0.lookback_time"); ok {
			t, _ := time.ParseDuration(v.(string))
			ruleConfig.FacetRule.LookbackTime = &t
		}
	}

	if data.Get("rule.0.threshold.#") == 1 {
		ruleConfig.ThresholdRule = &observe.MonitorRuleThresholdConfig{}

		v := data.Get("rule.0.threshold.0.compare_function")
		fn := observe.CompareFunction(toCamel(v.(string)))
		ruleConfig.ThresholdRule.CompareFunction = &fn

		if v, ok := data.GetOk("rule.0.threshold.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				ruleConfig.ThresholdRule.CompareValues = append(ruleConfig.ThresholdRule.CompareValues, i.(float64))
			}
		}
		if v, ok := data.GetOk("rule.0.threshold.0.lookback_time"); ok {
			t, _ := time.ParseDuration(v.(string))
			ruleConfig.ThresholdRule.LookbackTime = &t
		}
	}

	if data.Get("rule.0.promote.#") == 1 {
		ruleConfig.PromoteRule = &observe.MonitorRulePromoteConfig{}

		if v, ok := data.GetOk("rule.0.promote.0.primary_key"); ok {
			var values []string
			for _, el := range v.([]interface{}) {
				values = append(values, el.(string))
			}
			ruleConfig.PromoteRule.PrimaryKey = values
		}

		if v, ok := data.GetOk("rule.0.promote.0.kind_field"); ok {
			s := v.(string)
			ruleConfig.PromoteRule.KindField = &s
		}

		if v, ok := data.GetOk("rule.0.promote.0.description_field"); ok {
			s := v.(string)
			ruleConfig.PromoteRule.DescriptionField = &s
		}
	}

	return ruleConfig, nil
}

func newNotificationSpecConfig(data *schema.ResourceData) (notificationSpec *observe.NotificationSpecConfig, diags diag.Diagnostics) {
	var (
		defaultImportance = observe.NotificationImportance("Informational")
		defaultMerge      = observe.NotificationMerge("Merged")
	)

	notificationSpec = &observe.NotificationSpecConfig{
		Importance: &defaultImportance,
		Merge:      &defaultMerge,
	}

	if v, ok := data.GetOk("notification_spec.0.importance"); ok {
		s := observe.NotificationImportance(toCamel(v.(string)))
		notificationSpec.Importance = &s
	}

	if v, ok := data.GetOk("notification_spec.0.merge"); ok {
		s := observe.NotificationMerge(toCamel(v.(string)))
		notificationSpec.Merge = &s
	}

	return notificationSpec, nil
}

func newMonitorConfig(data *schema.ResourceData) (config *observe.MonitorConfig, diags diag.Diagnostics) {
	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, diags
	}

	if query == nil {
		return nil, diag.Errorf("no query provided")
	}

	rule, diags := newMonitorRuleConfig(data)
	if diags.HasError() {
		return nil, diags
	}

	notificationSpec, diags := newNotificationSpecConfig(data)
	if diags.HasError() {
		return nil, diags
	}

	config = &observe.MonitorConfig{
		Name:             data.Get("name").(string),
		Query:            query,
		Rule:             rule,
		Disabled:         data.Get("disabled").(bool),
		NotificationSpec: notificationSpec,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		s := v.(string)
		config.IconURL = &s
	}

	if v, ok := data.GetOk("freshness"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		config.Freshness = &t
	}

	if v, ok := data.GetOk("description"); ok {
		s := v.(string)
		config.Description = &s
	}

	return
}

func resourceMonitorCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitor(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create monitor: %s", err.Error())
	}

	data.SetId(result.ID)
	return append(diags, resourceMonitorRead(ctx, data, meta)...)
}

func resourceMonitorUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitor(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update monitor: %s", err.Error())
	}

	return append(diags, resourceMonitorRead(ctx, data, meta)...)
}

func resourceMonitorRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	monitor, err := client.GetMonitor(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read monitor: %s", err.Error())
	}

	workspaceOID := observe.OID{
		Type: observe.TypeWorkspace,
		ID:   monitor.WorkspaceID,
	}

	if err := data.Set("workspace", workspaceOID.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", monitor.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if monitor.Config.Freshness != nil {
		if err := data.Set("freshness", monitor.Config.Freshness.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("icon_url", monitor.Config.IconURL); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitor.Config.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("disabled", monitor.Config.Disabled); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("rule", flattenRule(monitor.Config.Rule)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("notification_spec", flattenNotificationSpec(monitor.Config.NotificationSpec)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := flattenAndSetQuery(data, monitor.Config.Query); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", monitor.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenRule(config *observe.MonitorRuleConfig) interface{} {
	rule := map[string]interface{}{
		"source_column": config.SourceColumn,
		"group_by":      toSnake(config.GroupBy.String()),
	}

	if len(config.GroupByGroups) > 0 {
		var list []interface{}
		for _, group := range config.GroupByGroups {
			list = append(list, map[string]interface{}{
				"group_name": group.GroupName,
				"columns":    group.Columns,
			})
		}
		rule["group_by_group"] = list
	}

	if config.ChangeRule != nil {
		change := map[string]interface{}{
			"change_type":        toSnake(config.ChangeRule.ChangeType.String()),
			"aggregate_function": toSnake(config.ChangeRule.AggregateFunction.String()),
			"compare_function":   toSnake(config.ChangeRule.CompareFunction.String()),
			"compare_values":     config.ChangeRule.CompareValues,
			"lookback_time":      config.ChangeRule.LookbackTime.String(),
			"baseline_time":      config.ChangeRule.BaselineTime.String(),
		}

		rule["change"] = []interface{}{change}
	}

	if config.CountRule != nil {
		count := map[string]interface{}{
			"compare_function": toSnake(config.CountRule.CompareFunction.String()),
			"compare_values":   config.CountRule.CompareValues,
			"lookback_time":    config.CountRule.LookbackTime.String(),
		}

		rule["count"] = []interface{}{count}
	}

	if config.FacetRule != nil {
		facet := map[string]interface{}{
			"facet_function": toSnake(config.FacetRule.FacetFunction.String()),
			"facet_values":   config.FacetRule.FacetValues,
			"time_function":  toSnake(config.FacetRule.TimeFunction.String()),
			"time_value":     config.FacetRule.TimeValue,
			"lookback_time":  config.FacetRule.LookbackTime.String(),
		}

		rule["facet"] = []interface{}{facet}
	}

	if config.ThresholdRule != nil {
		threshold := map[string]interface{}{
			"compare_function": toSnake(config.ThresholdRule.CompareFunction.String()),
			"compare_values":   config.ThresholdRule.CompareValues,
			"lookback_time":    config.ThresholdRule.LookbackTime.String(),
		}

		rule["threshold"] = []interface{}{threshold}
	}

	if config.PromoteRule != nil {
		promote := map[string]interface{}{
			"primary_key":       config.PromoteRule.PrimaryKey,
			"kind_field":        config.PromoteRule.KindField,
			"description_field": config.PromoteRule.DescriptionField,
		}

		rule["promote"] = []interface{}{promote}
	}

	return []interface{}{
		rule,
	}
}

func flattenNotificationSpec(config *observe.NotificationSpecConfig) interface{} {
	if config == nil {
		return nil
	}

	var results []interface{}

	results = append(results, map[string]interface{}{
		"merge":      toSnake(config.Merge.String()),
		"importance": toSnake(config.Importance.String()),
	})

	return results
}

func resourceMonitorDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitor(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor: %s", err.Error())
	}
	return diags
}
