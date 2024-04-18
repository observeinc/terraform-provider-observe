package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var validRules = []string{
	"rule.0.change",
	"rule.0.count",
	"rule.0.facet",
	"rule.0.threshold",
	"rule.0.promote",
	"rule.0.log",
}

func resourceMonitor() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("monitor", "description"),
		CreateContext: resourceMonitorCreate,
		ReadContext:   resourceMonitorRead,
		UpdateContext: resourceMonitorUpdate,
		DeleteContext: resourceMonitorDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitor", "schema", "name"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor", "schema", "description"),
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor", "schema", "comment"),
			},
			"freshness": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("transform", "schema", "freshness"),
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
				Description:      descriptions.Get("transform", "schema", "inputs"),
			},
			"is_template": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: descriptions.Get("monitor", "schema", "is_template"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: descriptions.Get("monitor", "schema", "disabled"),
			},
			"definition": {
				Type:             schema.TypeString,
				Default:          "{}",
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      descriptions.Get("monitor", "schema", "definition"),
			},
			"stage": {
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
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: diffSuppressPipeline,
							Description:      descriptions.Get("transform", "schema", "stage", "pipeline"),
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
			"rule": {
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
							Type:     schema.TypeList,
							Optional: true,
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
						"count": {
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
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
						"change": {
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
										ValidateDiagFunc: validateEnums(gql.AllChangeTypes),
									},
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
									},
									"aggregate_function": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "avg",
										ValidateDiagFunc: validateEnums(gql.AllAggregateFunctions),
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
						"facet": {
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"facet_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllFacetFunctions),
									},
									"facet_values": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 0,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"time_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllTimeFunctions),
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
						"threshold": {
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
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
									"threshold_agg_function": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "at_all_times",
										ValidateDiagFunc: validateEnums(gql.AllThresholdAggFunctions),
									},
								},
							},
						},
						"promote": {
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"primary_key": {
										Type:     schema.TypeList,
										Required: true,
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
						"log": {
							Type:         schema.TypeList,
							MaxItems:     1,
							Optional:     true,
							ExactlyOneOf: validRules,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
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
									"expression_summary": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: descriptions.Get("monitor", "schema", "rule", "log", "expression_summary"),
									},
									"log_stage_id": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: descriptions.Get("monitor", "schema", "rule", "log", "log_stage_id"),
									},
									"source_log_dataset": {
										Type:             schema.TypeString,
										Optional:         true,
										Description:      descriptions.Get("monitor", "schema", "rule", "log", "source_log_dataset"),
										ValidateDiagFunc: validateOID(oid.TypeDataset),
									},
								},
							},
						},
					},
				},
			},
			"notification_spec": {
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
							ValidateDiagFunc: validateEnums(gql.AllNotificationImportances),
						},
						"merge": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "merged",
							ValidateDiagFunc: validateEnums(gql.AllNotificationMerges),
						},
						"reminder_frequency": {
							Type:             schema.TypeString,
							Description:      descriptions.Get("monitor", "schema", "notification_spec", "reminder_frequency"),
							Optional:         true,
							ValidateDiagFunc: validateTimeDuration,
							DiffSuppressFunc: diffSuppressTimeDuration,
						},
						"notify_on_reminder": {
							Type:        schema.TypeBool,
							Description: descriptions.Get("monitor", "schema", "notification_spec", "notify_on_reminder"),
							Computed:    true,
						},
						"notify_on_close": {
							Type:        schema.TypeBool,
							Description: descriptions.Get("monitor", "schema", "notification_spec", "notify_on_close"),
							Optional:    true,
							Default:     false,
						},
					},
				},
			},
		},
	}
}

func newMonitorRuleConfig(data *schema.ResourceData) (ruleInput *gql.MonitorRuleInput, diags diag.Diagnostics) {
	ruleInput = &gql.MonitorRuleInput{
		GroupByGroups: make([]gql.MonitorGroupInfoInput, 0),
	}

	if v, ok := data.GetOk("rule.0.source_column"); ok {
		ruleInput.SourceColumn = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("rule.0.group_by_group"); ok {
		for _, el := range v.([]interface{}) {
			info := gql.MonitorGroupInfoInput{
				Columns: make([]string, 0),
			}
			if el != nil {
				value := el.(map[string]interface{})
				info.GroupName = value["group_name"].(string)
				for _, col := range value["columns"].([]interface{}) {
					info.Columns = append(info.Columns, col.(string))
				}
			}
			ruleInput.GroupByGroups = append(ruleInput.GroupByGroups, info)
		}
	}

	if data.Get("rule.0.count.#") == 1 {
		ruleInput.CountRule = &gql.MonitorRuleCountInput{}

		v := data.Get("rule.0.count.0.compare_function")
		fn := gql.CompareFunction(toCamel(v.(string)))
		ruleInput.CountRule.CompareFunction = &fn

		// TODO: remove compare_value
		if v, ok := data.GetOk("rule.0.count.0.compare_value"); ok {
			n := types.NumberScalar(v.(float64))
			ruleInput.CountRule.CompareValues = []types.NumberScalar{n}
		} else if v, ok := data.GetOk("rule.0.count.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				n := types.NumberScalar(i.(float64))
				ruleInput.CountRule.CompareValues = append(ruleInput.CountRule.CompareValues, n)
			}
		}
		if v, ok := data.GetOk("rule.0.count.0.lookback_time"); ok {
			lookbackTimeParsed, _ := types.ParseDurationScalar(v.(string))
			ruleInput.CountRule.LookbackTime = lookbackTimeParsed
		}
	}

	if data.Get("rule.0.change.#") == 1 {
		ruleInput.ChangeRule = &gql.MonitorRuleChangeInput{}

		if v := data.Get("rule.0.change.0.change_type"); true {
			changeType := gql.ChangeType(toCamel(v.(string)))
			ruleInput.ChangeRule.ChangeType = &changeType
		}

		if v := data.Get("rule.0.change.0.aggregate_function"); true {
			fn := gql.AggregateFunction(toCamel(v.(string)))
			ruleInput.ChangeRule.AggregateFunction = &fn
		}

		if v := data.Get("rule.0.change.0.compare_function"); true {
			fn := gql.CompareFunction(toCamel(v.(string)))
			ruleInput.ChangeRule.CompareFunction = &fn
		}

		// TODO: remove compare_value
		if v, ok := data.GetOk("rule.0.change.0.compare_value"); ok {
			n := types.NumberScalar(v.(float64))
			ruleInput.ChangeRule.CompareValues = []types.NumberScalar{n}
		} else if v, ok := data.GetOk("rule.0.change.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				n := types.NumberScalar(i.(float64))
				ruleInput.ChangeRule.CompareValues = append(ruleInput.ChangeRule.CompareValues, n)
			}
		}

		if v, ok := data.GetOk("rule.0.change.0.lookback_time"); ok {
			lookbackTimeParsed, _ := types.ParseDurationScalar(v.(string))
			ruleInput.ChangeRule.LookbackTime = lookbackTimeParsed
		}

		if v, ok := data.GetOk("rule.0.change.0.baseline_time"); ok {
			baselineTimeParsed, _ := types.ParseDurationScalar(v.(string))
			ruleInput.ChangeRule.BaselineTime = baselineTimeParsed
		}
	}

	if data.Get("rule.0.facet.#") == 1 {
		ruleInput.FacetRule = &gql.MonitorRuleFacetInput{}

		if v, ok := data.GetOk("rule.0.facet.0.facet_function"); ok {
			fn := gql.FacetFunction(toCamel(v.(string)))
			ruleInput.FacetRule.FacetFunction = &fn
		}

		if v, ok := data.GetOk("rule.0.facet.0.facet_values"); ok {
			var values []string
			for _, el := range v.([]interface{}) {
				values = append(values, el.(string))
			}
			ruleInput.FacetRule.FacetValues = values
		}

		if v, ok := data.GetOk("rule.0.facet.0.time_function"); ok {
			fn := gql.TimeFunction(toCamel(v.(string)))
			ruleInput.FacetRule.TimeFunction = &fn
		}

		if v, ok := data.GetOk("rule.0.facet.0.time_value"); ok {
			f := types.NumberScalar(v.(float64))
			ruleInput.FacetRule.TimeValue = &f
		}

		if v, ok := data.GetOk("rule.0.facet.0.lookback_time"); ok {
			parsedLookbackTime, _ := types.ParseDurationScalar(v.(string))
			ruleInput.FacetRule.LookbackTime = parsedLookbackTime
		}
	}

	if data.Get("rule.0.threshold.#") == 1 {
		ruleInput.ThresholdRule = &gql.MonitorRuleThresholdInput{}

		v := data.Get("rule.0.threshold.0.compare_function")
		fn := gql.CompareFunction(toCamel(v.(string)))
		ruleInput.ThresholdRule.CompareFunction = &fn

		if v, ok := data.GetOk("rule.0.threshold.0.threshold_agg_function"); ok {
			aggFn := gql.ThresholdAggFunction(toCamel(v.(string)))
			ruleInput.ThresholdRule.ThresholdAggFunction = &aggFn
		}

		if v, ok := data.GetOk("rule.0.threshold.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				n := types.NumberScalar(i.(float64))
				ruleInput.ThresholdRule.CompareValues = append(ruleInput.ThresholdRule.CompareValues, n)
			}
		}
		if v, ok := data.GetOk("rule.0.threshold.0.lookback_time"); ok {
			parsedLookbackTime, _ := types.ParseDurationScalar(v.(string))
			ruleInput.ThresholdRule.LookbackTime = parsedLookbackTime
		}
	}

	if data.Get("rule.0.promote.#") == 1 {
		ruleInput.PromoteRule = &gql.MonitorRulePromoteInput{}

		if v, ok := data.GetOk("rule.0.promote.0.primary_key"); ok {
			var values []string
			for _, el := range v.([]interface{}) {
				values = append(values, el.(string))
			}
			ruleInput.PromoteRule.PrimaryKey = values
		}

		if v, ok := data.GetOk("rule.0.promote.0.kind_field"); ok {
			s := v.(string)
			ruleInput.PromoteRule.KindField = &s
		}

		if v, ok := data.GetOk("rule.0.promote.0.description_field"); ok {
			s := v.(string)
			ruleInput.PromoteRule.DescriptionField = &s
		}
	}

	if data.Get("rule.0.log.#") == 1 {
		ruleInput.LogRule = &gql.MonitorRuleLogInput{}

		v := data.Get("rule.0.log.0.compare_function")
		fn := gql.CompareFunction(toCamel(v.(string)))
		ruleInput.LogRule.CompareFunction = &fn

		if v, ok := data.GetOk("rule.0.log.0.compare_values"); ok {
			for _, i := range v.([]interface{}) {
				n := types.NumberScalar(i.(float64))
				ruleInput.LogRule.CompareValues = append(ruleInput.LogRule.CompareValues, n)
			}
		}

		if v, ok := data.GetOk("rule.0.log.0.lookback_time"); ok {
			parsedLookbackTime, _ := types.ParseDurationScalar(v.(string))
			ruleInput.LogRule.LookbackTime = parsedLookbackTime
		}

		if v, ok := data.GetOk("rule.0.log.0.expression_summary"); ok {
			s := v.(string)
			ruleInput.LogRule.ExpressionSummary = &s
		}

		if v, ok := data.GetOk("rule.0.log.0.log_stage_id"); ok {
			s := v.(string)
			ruleInput.LogRule.LogStageId = &s
		}

		if v, ok := data.GetOk("rule.0.log.0.source_log_dataset"); ok {
			is, _ := oid.NewOID(v.(string))
			ruleInput.LogRule.SourceLogDatasetId = &is.Id
		}
	}

	return ruleInput, nil
}

func newNotificationSpecConfig(data *schema.ResourceData) (notificationSpec *gql.NotificationSpecificationInput, diags diag.Diagnostics) {
	var (
		defaultImportance = gql.NotificationImportance("Informational")
		defaultMerge      = gql.NotificationMerge("Merged")
	)

	notificationSpec = &gql.NotificationSpecificationInput{
		Importance: &defaultImportance,
		Merge:      &defaultMerge,
	}

	if v, ok := data.GetOk("notification_spec.0.reminder_frequency"); ok {
		d, _ := types.ParseDurationScalar(v.(string))
		notificationSpec.ReminderFrequency = d
		notificationSpec.NotifyOnReminder = boolPtr(*d != 0)
	}

	if v, ok := data.GetOk("notification_spec.0.notify_on_close"); ok {
		notificationSpec.NotifyOnClose = boolPtr(v.(bool))
	}

	if v, ok := data.GetOk("notification_spec.0.importance"); ok {
		s := gql.NotificationImportance(toCamel(v.(string)))
		notificationSpec.Importance = &s
	}

	if v, ok := data.GetOk("notification_spec.0.merge"); ok {
		s := gql.NotificationMerge(toCamel(v.(string)))
		notificationSpec.Merge = &s
	}

	return notificationSpec, nil
}

func newMonitorConfig(data *schema.ResourceData) (input *gql.MonitorInput, diags diag.Diagnostics) {
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

	name := data.Get("name").(string)
	disabled := data.Get("disabled").(bool)
	isTemplate := data.Get("is_template").(bool)

	overwriteSource := true
	input = &gql.MonitorInput{
		Name:             &name,
		Query:            query,
		Rule:             rule,
		Disabled:         &disabled,
		IsTemplate:       &isTemplate,
		NotificationSpec: notificationSpec,
		OverwriteSource:  &overwriteSource,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("freshness"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		input.FreshnessGoal = types.Int64Scalar(t).Ptr()
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("comment"); ok {
		input.Comment = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("definition"); ok {
		input.Definition = types.JsonObject(v.(string)).Ptr()
	}

	return
}

func resourceMonitorCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitor(ctx, id.Id, config)
	if err != nil {
		return diag.Errorf("failed to create monitor: %s", err.Error())
	}

	data.SetId(result.Id)
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

	if err := data.Set("workspace", oid.WorkspaceOid(monitor.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", monitor.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if !monitor.UseDefaultFreshness {
		if err := data.Set("freshness", monitor.FreshnessGoal.Duration().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("icon_url", monitor.IconUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitor.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if monitor.Comment != nil {
		if err := data.Set("comment", *monitor.Comment); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("is_template", monitor.IsTemplate); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("disabled", monitor.Disabled); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("notification_spec", flattenNotificationSpec(monitor.NotificationSpec)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	stageIds, err := flattenAndSetQuery(data, monitor.Query.Stages, monitor.Query.OutputStage)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("rule", flattenRule(data, monitor.Rule, stageIds)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if monitor.Definition != nil {
		if err := data.Set("definition", monitor.Definition); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", monitor.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenRule(data *schema.ResourceData, input gql.MonitorRule, stageIds []string) interface{} {
	rule := map[string]interface{}{
		"source_column": input.GetSourceColumn(),
	}

	var list []interface{}
	for _, group := range input.GetGroupByGroups() {
		list = append(list, map[string]interface{}{
			"group_name": group.GroupName,
			"columns":    group.Columns,
		})
	}
	rule["group_by_group"] = list

	if changeRule, ok := input.(*gql.MonitorRuleMonitorRuleChange); ok {
		change := map[string]interface{}{
			"change_type":        toSnake(string(changeRule.ChangeType)),
			"aggregate_function": toSnake(string(changeRule.AggregateFunction)),
			"compare_function":   toSnake(string(changeRule.CompareFunction)),
			"compare_values":     changeRule.CompareValues,
			"lookback_time":      changeRule.LookbackTime.String(),
			"baseline_time":      changeRule.BaselineTime.String(),
		}

		rule["change"] = []interface{}{change}
	}

	if countRule, ok := input.(*gql.MonitorRuleMonitorRuleCount); ok {
		count := map[string]interface{}{
			"compare_function": toSnake(string(countRule.CompareFunction)),
			"compare_values":   countRule.CompareValues,
			"lookback_time":    countRule.LookbackTime.String(),
		}

		rule["count"] = []interface{}{count}
	}

	if facetRule, ok := input.(*gql.MonitorRuleMonitorRuleFacet); ok {
		facet := map[string]interface{}{
			"facet_function": toSnake(string(facetRule.FacetFunction)),
			"facet_values":   facetRule.FacetValues,
			"time_function":  toSnake(string(facetRule.TimeFunction)),
			"time_value":     facetRule.TimeValue,
			"lookback_time":  facetRule.LookbackTime.String(),
		}

		rule["facet"] = []interface{}{facet}
	}

	if thresholdRule, ok := input.(*gql.MonitorRuleMonitorRuleThreshold); ok {
		threshold := map[string]interface{}{
			"compare_function":       toSnake(string(thresholdRule.CompareFunction)),
			"compare_values":         thresholdRule.CompareValues,
			"lookback_time":          thresholdRule.LookbackTime.String(),
			"threshold_agg_function": toSnake(string(thresholdRule.ThresholdAggFunction)),
		}

		rule["threshold"] = []interface{}{threshold}
	}

	if promoteRule, ok := input.(*gql.MonitorRuleMonitorRulePromote); ok {
		promote := map[string]interface{}{
			"primary_key":       promoteRule.PrimaryKey,
			"kind_field":        promoteRule.KindField,
			"description_field": promoteRule.DescriptionField,
		}

		rule["promote"] = []interface{}{promote}
	}

	if logRule, ok := input.(*gql.MonitorRuleMonitorRuleLog); ok {
		log := map[string]interface{}{
			"compare_function":   toSnake(string(logRule.CompareFunction)),
			"compare_values":     logRule.CompareValues,
			"lookback_time":      logRule.LookbackTime.String(),
			"expression_summary": logRule.ExpressionSummary,
		}

		for i, sId := range stageIds {
			if sId == logRule.LogStageId {
				// We can update the previous logStageId value to the new format one but pointing to the same stage
				log["log_stage_id"] = fmt.Sprintf("stage-%d", i)
				break
			}
		}

		if logRule.SourceLogDatasetId != nil {
			id := oid.DatasetOid(*logRule.SourceLogDatasetId)
			// check for existing version timestamp we can maintain
			// same approach as in flattenAndSetQuery() for input datasets
			if v, ok := data.GetOk("rule.0.log.0.source_log_dataset"); ok {
				prv, err := oid.NewOID(v.(string))
				if err == nil && id.Id == prv.Id {
					id.Version = prv.Version
				}
			}
			log["source_log_dataset"] = id.String()
		}
		rule["log"] = []interface{}{log}
	}

	return []interface{}{
		rule,
	}
}

func flattenNotificationSpec(spec gql.MonitorNotificationSpecNotificationSpecification) interface{} {
	result := map[string]interface{}{
		"merge":      toSnake(string(*spec.Merge)),
		"importance": toSnake(string(spec.Importance)),
	}

	if spec.NotifyOnReminder != nil {
		result["notify_on_reminder"] = *spec.NotifyOnReminder

		if *spec.NotifyOnReminder && spec.ReminderFrequency != 0 {
			result["reminder_frequency"] = spec.ReminderFrequency.String()
		}
	}

	if spec.NotifyOnClose != nil {
		result["notify_on_close"] = *spec.NotifyOnClose
	}

	return []interface{}{result}
}

func resourceMonitorDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitor(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor: %s", err.Error())
	}
	return diags
}
