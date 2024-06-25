package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// TODO: make the schema keys constants?
// annoying to change varnames in 3 non-obvious places

func resourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2Create,
		ReadContext:   resourceMonitorV2Read,
		UpdateContext: resourceMonitorV2Update,
		DeleteContext: resourceMonitorV2Delete,
		Schema: map[string]*schema.Schema{
			"comment": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"rule_kind": { // MonitorV2RuleKind!
				Type:     schema.TypeString,
				Required: true,
			},
			"name": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"managed_by_id": { // ObjectId
				Type:     schema.TypeString,
				Optional: true,
			},
			"folder_id": { // ObjectId
				Type:     schema.TypeString,
				Optional: true,
			},
			"stage": { // for building inputQuery (MultiStageQueryInput!)
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
							Optional: true,
						},
						"output_stage": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},
					},
				},
			},
			"rules": { // [MonitorV2RuleInput!]!
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{ // MonitorV2RuleInput!
						"level": { // MonitorV2AlarmLevel!
							Type:     schema.TypeString,
							Required: true,
						},
						"count": { // [MonitorV2ComparisonInput!]
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{ // MonitorV2ComparisonInput
									"compare_fn": { // MonitorV2ComparisonFunction!
										//   Equal
										//   Greater
										//   GreaterOrEqual
										//   Less
										//   LessOrEqual
										//   NotEqual
										Type:     schema.TypeString,
										Required: true,
									},
									"compare_value": { // PrimitiveValueInput!
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										MinItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"bool": { // Boolean
													Type:     schema.TypeBool,
													Optional: true,
												},
												"float64": { // Float
													Type:     schema.TypeFloat,
													Optional: true,
												},
												"int64": { // Int64
													Type:     schema.TypeInt,
													Optional: true,
												},
												"string": { // String
													Type:     schema.TypeString,
													Optional: true,
												},
												"timestamp": { // Time
													Type:     schema.TypeString,
													Optional: true,
												},
												"duration": { // Int64
													Type:     schema.TypeInt,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						"threshold": { // MonitorV2ThresholdRuleInput
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"compare_fn": { // MonitorV2ComparisonFunction!
													//   Equal
													//   Greater
													//   GreaterOrEqual
													//   Less
													//   LessOrEqual
													//   NotEqual
													Type:     schema.TypeString,
													Required: true,
												},
												"compare_value": { // PrimitiveValueInput!
													// PrimitiveValue
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"bool": { // Boolean
																Type:     schema.TypeBool,
																Optional: true,
															},
															"float64": { // Float
																Type:     schema.TypeFloat,
																Optional: true,
															},
															"int64": { // Int64
																Type:     schema.TypeInt,
																Optional: true,
															},
															"string": { // String
																Type:     schema.TypeString,
																Optional: true,
															},
															"timestamp": { // Time
																Type:     schema.TypeString,
																Optional: true,
															},
															"duration": { // Int64
																Type:     schema.TypeInt,
																Optional: true,
															},
														},
													},
												},
											},
										},
									},
									"value_column_name": { // String!
										Type:     schema.TypeString,
										Required: true,
									},
									"aggregation": { // MonitorV2ValueAggregation!
										//   AllOf
										//   AnyOf
										//   AvgOf
										//   SumOf
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"promote": { // [MonitorV2ColumnComparisonInput!] (aka MonitorV2PromoteRuleInput)
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{ // MonitorV2ComparisonInput!
												"compare_fn": { // MonitorV2ComparisonFunction!
													//   Equal
													//   Greater
													//   GreaterOrEqual
													//   Less
													//   LessOrEqual
													//   NotEqual
													Type:     schema.TypeString,
													Required: true,
												},
												"compare_value": { // PrimitiveValueInput!
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"bool": { // Boolean
																Type:     schema.TypeBool,
																Optional: true,
															},
															"float64": { // Float
																Type:     schema.TypeFloat,
																Optional: true,
															},
															"int64": { // Int64
																Type:     schema.TypeInt,
																Optional: true,
															},
															"string": { // String
																Type:     schema.TypeString,
																Optional: true,
															},
															"timestamp": { // Time
																Type:     schema.TypeString,
																Optional: true,
															},
															"duration": { // Int64
																Type:     schema.TypeInt,
																Optional: true,
															},
														},
													},
												},
											},
										},
									},
									"column": { // MonitorV2ColumnInput!
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{ // MonitorV2ColumnInput
												"link_column": { // MonitorV2LinkColumnInput
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 1,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": { // String!
																Type:     schema.TypeString,
																Required: true,
															},
															"meta": { // MonitorV2LinkColumnMetaInput
																Type:     schema.TypeList,
																Optional: true,
																MinItems: 1,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"src_fields": { // [MonitorV2ColumnPathInput!]
																			Type:     schema.TypeList,
																			Optional: true,
																			MinItems: 1,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{},
																			},
																		},
																		"dst_fields": { // [String!]
																			Type:     schema.TypeList,
																			Optional: true,
																			MinItems: 1,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																		"target_dataset": { // Int64
																			Type:     schema.TypeInt,
																			Optional: true,
																		},
																	},
																},
															},
														},
													},
												},
												"column_path": { // MonitorV2ColumnPathInput
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 1,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": { // String!
																Type:     schema.TypeString,
																Required: true,
															},
															"path": { // String
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
											},
										},
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
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"data_stabilization_delay": { // Duration
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"groupings": { // [MonitorV2ColumnInput!]
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"link_column": { // MonitorV2LinkColumnInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": { // String!
										Type:     schema.TypeString,
										Required: true,
									},
									"meta": { // MonitorV2LinkColumnMetaInput
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"src_fields": { // [MonitorV2ColumnPathInput!]
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": { // String!
																Type:     schema.TypeString,
																Required: true,
															},
															"path": { // String
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"dst_fields": { // [String!]
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
												"target_dataset": { // Int64
													Type:     schema.TypeInt,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						"column_path": { // MonitorV2ColumnPathInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": { // String!
										Type:     schema.TypeString,
										Required: true,
									},
									"path": { // String
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"scheduling": { // MonitorV2SchedulingInput
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": { // MonitorV2IntervalScheduleInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"interval": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDuration,
									},
									"randomize": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDuration,
									},
								},
							},
						},
						"transform": { // Duration (MonitorV2TransformScheduleInput)
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validateTimeDuration,
							DiffSuppressFunc: diffSuppressTimeDuration,
						},
					},
				},
			},
		},
	}
}

func newMonitorV2CountRuleInput(data *schema.ResourceData, i int) *gql.MonitorV2CountRuleInput {
	var countRule gql.MonitorV2CountRuleInput

	if v, ok := data.GetOk(fmt.Sprintf("rules.%d.count.0.compare_fn", i)); ok {
		countRule.CompareFn = gql.MonitorV2ComparisonFunction(v.(string))
	} else {
		return nil
	}

	if v, ok := data.GetOk(fmt.Sprintf("rules.%d.count.0.compare_value", i)); ok {
		countRule.CompareValue = types.NumberScalar(v.(float64))
	} else {
		return nil
	}

	return &countRule
}

func newMonitorV2Rules(data *schema.ResourceData) (rules []gql.MonitorV2RuleInput, diags diag.Diagnostics) {
	rules = make([]gql.MonitorV2RuleInput, 0)
	for i := range data.Get("rules").([]interface{}) {
		var rule gql.MonitorV2RuleInput
		if v, ok := data.GetOk(fmt.Sprintf("rules.%d.level", i)); ok {
			rule.Level = gql.MonitorV2AlarmLevel(v.(string))
		}
		rule.Count = newMonitorV2CountRuleInput(data, i)
		rules = append(rules, rule)
	}
	return rules, diags
}

func newMonitorV2GroupByGroups(data *schema.ResourceData) (groupByGroups []gql.MonitorGroupInfoInput, diags diag.Diagnostics) {
	groupByGroups = make([]gql.MonitorGroupInfoInput, 0)
	for i := range data.Get("group_by_groups").([]interface{}) {
		var g gql.MonitorGroupInfoInput
		g.Columns = make([]string, 0)
		for j := range data.Get(fmt.Sprintf("group_by_groups.%d.columns", i)).([]interface{}) {
			if v, ok := data.GetOk(fmt.Sprintf("group_by_groups.%d.columns.%d", i, j)); ok {
				g.Columns = append(g.Columns, v.(string))
			}
		}
		if v, ok := data.GetOk(fmt.Sprintf("group_by_groups.%d.group_name", i)); ok {
			g.GroupName = v.(string)
		}
		var col string
		var path string
		if v, ok := data.GetOk(fmt.Sprintf("group_by_groups.%d.column_path.0.column", i)); ok {
			col = v.(string)
		}
		if v, ok := data.GetOk(fmt.Sprintf("group_by_groups.%d.column_path.0.path", i)); ok {
			path = v.(string)
		}
		g.ColumnPath = &gql.MonitorGroupByColumnPathInput{
			Column: col,
			Path:   path,
		}
		groupByGroups = append(groupByGroups, g)
	}
	return groupByGroups, diags
}

func newMonitorV2DefinitionInput(data *schema.ResourceData) (defnInput *gql.MonitorV2DefinitionInput, diags diag.Diagnostics) {
	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, diags
	}
	if query == nil {
		return nil, diag.Errorf("no query provided")
	}

	rules, diags := newMonitorV2Rules(data)
	if diags.HasError() {
		return nil, diags
	}

	groupByGroups, diags := newMonitorV2GroupByGroups(data)
	if diags.HasError() {
		return nil, diags
	}

	defnInput = &gql.MonitorV2DefinitionInput{
		InputQuery:    *query,
		Rules:         rules,
		GroupByGroups: groupByGroups,
	}

	// optionals
	if v, ok := data.GetOk("lookback_time"); ok {
		lookbackTime, _ := types.ParseDurationScalar(v.(string))
		defnInput.LookbackTime = lookbackTime
	}

	return defnInput, diags
}

func newMonitorV2Input(data *schema.ResourceData) (input *gql.MonitorV2Input, diags diag.Diagnostics) {
	// is this how to read non-optionals?
	disabled := data.Get("disabled").(bool)
	definitionInput, diags := newMonitorV2DefinitionInput(data)
	if diags.HasError() {
		return nil, diags
	}
	ruleKind := data.Get("rule_kind").(string)
	name := data.Get("name").(string)

	input = &gql.MonitorV2Input{
		Disabled:   disabled,
		Definition: *definitionInput,
		RuleKind:   meta.MonitorV2RuleKind(ruleKind),
		Name:       name,
	}

	// optionals
	if v, ok := data.GetOk("comment"); ok {
		input.Comment = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("managed_by_id"); ok {
		input.ManagedById = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("folder_id"); ok {
		input.FolderId = stringPtr(v.(string))
	}

	return input, diags
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
		return diag.Errorf("failed to create monitor: %s", err.Error())
	}

	return append(diags, resourceMonitorV2Read(ctx, data, meta)...)
}

func resourceMonitorV2Read(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	monitor, err := client.GetMonitorV2(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read monitor: %s", err.Error())
	}

	// perform data.set on all the fields from this monitor
	if err := data.Set("name", monitor.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
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
