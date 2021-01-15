package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

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
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
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
							Type:     schema.TypeString,
							Required: true,
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
						"group_by": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "none",
							ValidateDiagFunc: validateEnums(observe.MonitorGroupings),
						},
						"group_by_columns": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"count": &schema.Schema{
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(observe.CompareFunctions),
									},
									"value": {
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
						"change": &schema.Schema{
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: []string{"rule.0.change", "rule.0.count"},
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
									"value": {
										Type:         schema.TypeFloat,
										Optional:     true,
										ExactlyOneOf: []string{"rule.0.change.0.value", "rule.0.change.0.values"},
									},
									"values": {
										Type:         schema.TypeList,
										Optional:     true,
										MinItems:     2,
										MaxItems:     2,
										Elem:         &schema.Schema{Type: schema.TypeFloat},
										ExactlyOneOf: []string{"rule.0.change.0.value", "rule.0.change.0.values"},
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
						"selection": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "any",
							ValidateDiagFunc: validateEnums(observe.NotificationSelections),
						},
						"selection_value": {
							Type:     schema.TypeFloat,
							Optional: true,
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

	if v, ok := data.GetOk("rule.0.group_by_columns"); ok {
		for _, el := range v.([]interface{}) {
			ruleConfig.GroupByColumns = append(ruleConfig.GroupByColumns, el.(string))
		}
	}

	if data.Get("rule.0.count.#") == 1 {
		ruleConfig.CountRule = &observe.MonitorRuleCountConfig{}

		v := data.Get("rule.0.count.0.compare_function")
		fn := observe.CompareFunction(toCamel(v.(string)))
		ruleConfig.CountRule.CompareFunction = &fn

		if v, ok := data.GetOk("rule.0.count.0.value"); ok {
			ruleConfig.CountRule.CompareValues = []float64{v.(float64)}
		} else if v, ok := data.GetOk("rule.0.count.0.values"); ok {
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

		if v, ok := data.GetOk("rule.0.change.0.value"); ok {
			ruleConfig.ChangeRule.CompareValues = []float64{v.(float64)}
		} else if v, ok := data.GetOk("rule.0.change.0.values"); ok {
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
	return ruleConfig, nil
}

func newNotificationSpecConfig(data *schema.ResourceData) (notificationSpec *observe.NotificationSpecConfig, diags diag.Diagnostics) {
	var (
		defaultImportance = observe.NotificationImportance("Informational")
		defaultMerge      = observe.NotificationMerge("Merged")
		defaultSelection  = observe.NotificationSelection("Any")
	)

	notificationSpec = &observe.NotificationSpecConfig{
		Importance: &defaultImportance,
		Merge:      &defaultMerge,
		Selection:  &defaultSelection,
	}

	if v, ok := data.GetOk("notification_spec.0.importance"); ok {
		s := observe.NotificationImportance(toCamel(v.(string)))
		notificationSpec.Importance = &s
	}

	if v, ok := data.GetOk("notification_spec.0.merge"); ok {
		s := observe.NotificationMerge(toCamel(v.(string)))
		notificationSpec.Merge = &s
	}

	if v, ok := data.GetOk("notification_spec.0.selection"); ok {
		s := observe.NotificationSelection(toCamel(v.(string)))
		notificationSpec.Selection = &s
	}

	if v, ok := data.GetOk("notification_spec.0.selection_value"); ok {
		f := v.(float64)
		notificationSpec.SelectionValue = &f
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
		NotificationSpec: notificationSpec,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		s := v.(string)
		config.IconURL = &s
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

	if err := data.Set("icon_url", monitor.Config.IconURL); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitor.Config.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("rule", flattenRule(monitor.Config.Rule)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("notification_spec", flattenNotificationSpec(monitor.Config.NotificationSpec)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", monitor.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenRule(config *observe.MonitorRuleConfig) interface{} {
	rule := map[string]interface{}{
		"source_column":    config.SourceColumn,
		"group_by":         toSnake(config.GroupBy.String()),
		"group_by_columns": config.GroupByColumns,
	}

	if config.ChangeRule != nil {
		change := map[string]interface{}{
			"change_type":        toSnake(config.ChangeRule.ChangeType.String()),
			"aggregate_function": toSnake(config.ChangeRule.AggregateFunction.String()),
			"compare_function":   toSnake(config.ChangeRule.CompareFunction.String()),
			"lookback_time":      config.ChangeRule.LookbackTime.String(),
			"baseline_time":      config.ChangeRule.BaselineTime.String(),
		}

		switch len(config.ChangeRule.CompareValues) {
		case 1:
			change["value"] = config.ChangeRule.CompareValues[0]
		case 2:
			change["values"] = config.ChangeRule.CompareValues
		}

		rule["change"] = []interface{}{change}
	}

	if config.CountRule != nil {
		count := map[string]interface{}{
			"compare_function": toSnake(config.CountRule.CompareFunction.String()),
			"lookback_time":    config.CountRule.LookbackTime.String(),
		}

		switch len(config.CountRule.CompareValues) {
		case 1:
			count["value"] = config.CountRule.CompareValues[0]
		case 2:
			count["values"] = config.CountRule.CompareValues
		}

		rule["count"] = []interface{}{count}
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
		"merge":           toSnake(config.Merge.String()),
		"importance":      toSnake(config.Importance.String()),
		"selection":       toSnake(config.Selection.String()),
		"selection_value": config.SelectionValue,
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
