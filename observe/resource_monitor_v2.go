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
			// fields of MonitorV2Input excluding the components of MonitorV2DefinitionInput
			"workspace_id": { // ObjectId!
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"comment": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"rule_kind": { // MonitorV2RuleKind!
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2RuleKinds),
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
				Type:             schema.TypeString,
				ValidateDiagFunc: validateOID(oid.TypeUser),
				Optional:         true,
			},
			"folder_id": { // ObjectId
				Type:             schema.TypeString,
				ValidateDiagFunc: validateOID(oid.TypeFolder),
				Optional:         true,
			},
			// until specified otherwise, the following are for building MonitorV2DefinitionInput
			"stage": { // for building inputQuery (MultiStageQueryInput!))
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
			"inputs": { // for building inputQuery (MultiStageQueryInput!)
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
			},
			"rules": { // [MonitorV2RuleInput!]!
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{ // MonitorV2RuleInput!
						"level": { // MonitorV2AlarmLevel!
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateEnums(gql.AllMonitorV2AlarmLevels),
						},
						"count": { // MonitorV2CountRuleInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{ // MonitorV2ComparisonInput
												"compare_fn": { // MonitorV2ComparisonFunction!
													Type:             schema.TypeString,
													Required:         true,
													ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
												},
												"value_int64": { // Int64
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeInt},
												},
												"value_float64": { // Float
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeFloat},
												},
												"value_bool": { // Boolean
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeBool},
												},
												"value_string": { // String
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
												"value_duration": { // Int64
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Schema{
														Type:             schema.TypeBool,
														ValidateDiagFunc: validateTimeDuration,
														DiffSuppressFunc: diffSuppressDuration,
													},
												},
												"value_timestamp": { // Time
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Schema{
														Type:             schema.TypeString,
														ValidateDiagFunc: validateTimestamp,
													},
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
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_values": { // [MonitorV2ComparisonInput!]!
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"compare_fn": { // MonitorV2ComparisonFunction!
													Type:             schema.TypeString,
													Required:         true,
													ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
												},
												"value_int64": { // Int64
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeInt},
												},
												"value_float64": { // Float
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeFloat},
												},
												"value_bool": { // Boolean
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeBool},
												},
												"value_string": { // String
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
												"value_duration": { // Int64
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Schema{
														Type:             schema.TypeBool,
														ValidateDiagFunc: validateTimeDuration,
														DiffSuppressFunc: diffSuppressDuration,
													},
												},
												"value_timestamp": { // Time
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													MinItems: 1,
													Elem: &schema.Schema{
														Type:             schema.TypeString,
														ValidateDiagFunc: validateTimestamp,
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
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateEnums(gql.AllMonitorV2ValueAggregations),
									},
								},
							},
						},
						"promote": { // MonitorV2PromoteRuleInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_columns": { // [MonitorV2ColumnComparisonInput!]
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
																Type:             schema.TypeString,
																Required:         true,
																ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
															},
															"value_int64": { // Int64
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem:     &schema.Schema{Type: schema.TypeInt},
															},
															"value_float64": { // Float
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem:     &schema.Schema{Type: schema.TypeFloat},
															},
															"value_bool": { // Boolean
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem:     &schema.Schema{Type: schema.TypeBool},
															},
															"value_string": { // String
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
															"value_duration": { // Int64
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem: &schema.Schema{
																	Type:             schema.TypeBool,
																	ValidateDiagFunc: validateTimeDuration,
																	DiffSuppressFunc: diffSuppressDuration,
																},
															},
															"value_timestamp": { // Time
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																MinItems: 1,
																Elem: &schema.Schema{
																	Type:             schema.TypeString,
																	ValidateDiagFunc: validateTimestamp,
																},
															},
														},
													},
												},
												"column": {
													Type:     schema.TypeList,
													Required: true,
													MinItems: 1,
													MaxItems: 1,
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
				Required:         true,
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
									"src_fields": { // MonitorV2LinkColumnMetaInput.[MonitorV2ColumnPathInput!]
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
									"dst_fields": { // MonitorV2LinkColumnMetaInput.[String!]
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"target_dataset": { // MonitorV2LinkColumnMetaInput.Int64
										Type:     schema.TypeInt,
										Optional: true,
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
				MaxItems: 1,
				MinItems: 1,
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
						"transform": { // MonitorV2TransformScheduleInput
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"freshness_goal": { // Duration!
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateTimeDuration,
										DiffSuppressFunc: diffSuppressTimeDuration,
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

func resourceMonitorV2Create(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2Input(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace_id").(string))
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
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2Create(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to create monitor: %s", err.Error())
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

	_, err = flattenAndSetQuery(data, monitor.Definition.InputQuery.Stages, monitor.Definition.InputQuery.OutputStage)
	if err != nil {
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

func newMonitorV2Input(data *schema.ResourceData) (input *gql.MonitorV2Input, diags diag.Diagnostics) {
	// required
	definitionInput, diags := newMonitorV2DefinitionInput(data)
	if diags.HasError() {
		return nil, diags
	}
	ruleKind := data.Get("rule_kind").(string)
	name := data.Get("name").(string)

	// instantiation
	input = &gql.MonitorV2Input{
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
	lookbackTime, err := types.ParseDurationScalar(data.Get("lookback_time").(string))
	if err != nil {
		return nil, diag.Errorf("lookback_time is invalid: %s", err.Error())
	}

	// instantiation
	defnInput = &gql.MonitorV2DefinitionInput{
		InputQuery:   *query,
		Rules:        rules,
		LookbackTime: *lookbackTime,
	}

	// optionals
	if v, ok := data.GetOk("data_stabilization_delay"); ok {
		dataStabilizationDelay, _ := types.ParseDurationScalar(v.(string))
		defnInput.DataStabilizationDelay = dataStabilizationDelay
	}
	if _, ok := data.GetOk("groupings"); ok {
		groupings := make([]gql.MonitorV2ColumnInput, 0)
		for _, i := range data.Get("groupings").([]interface{}) {
			colInput, diags := newMonitorV2ColumnInput(fmt.Sprintf("groupings.%d.", i), data)
			if diags.HasError() {
				return nil, diags
			}
			groupings = append(groupings, *colInput)
		}
		defnInput.Groupings = groupings
	}
	if _, ok := data.GetOk("scheduling"); ok {
		scheduling, diags := newMonitorV2SchedulingInput("scheduling.0.", data)
		if diags.HasError() {
			return nil, diags
		}
		defnInput.Scheduling = scheduling
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

	return scheduling, diags
}

func newMonitorV2IntervalScheduleInput(path string, data *schema.ResourceData) (interval *gql.MonitorV2IntervalScheduleInput, diags diag.Diagnostics) {
	// required
	intervalField := data.Get(fmt.Sprintf("%sinterval", path)).(string)
	intervalDuration, _ := types.ParseDurationScalar(intervalField)
	randomizeField := data.Get(fmt.Sprintf("%srandomize", path)).(string)
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
	level := data.Get(fmt.Sprintf("%slevel", path)).(string)

	// instantiation
	rule = &gql.MonitorV2RuleInput{Level: gql.MonitorV2AlarmLevel(level)}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%scount", path)); ok {
		count, diags := newMonitorV2CountRuleInput(fmt.Sprintf("%scount.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		rule.Count = count
	}
	if _, ok := data.GetOk(fmt.Sprintf("%sthreshold", path)); ok {
		threshold, diags := newMonitorV2ThresholdRuleInput(fmt.Sprintf("%s.threshold.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		rule.Threshold = threshold
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

	return comparison, diags
}

func newMonitorV2ComparisonInput(path string, data *schema.ResourceData) (comparison *gql.MonitorV2ComparisonInput, diags diag.Diagnostics) {
	// required
	compareFn := gql.MonitorV2ComparisonFunction(data.Get(fmt.Sprintf("%scompare_fn", path)).(string))
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
	aggregation := gql.MonitorV2ValueAggregation(data.Get(fmt.Sprintf("%scompare_fn", path)).(string))

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
			input, diags := newMonitorV2ColumnComparisonInput(fmt.Sprintf("%s.compare_columns.%d.", prefix, i), data)
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
	columnInput, diags := newMonitorV2ColumnInput(fmt.Sprintf("%scolumn.", path), data)
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
	// required
	linkColumnInput, diags := newMonitorV2LinkColumnInput(fmt.Sprintf("%slink_column.0.", path), data)
	if diags.HasError() {
		return nil, diags
	}

	// instantiation
	column = &gql.MonitorV2ColumnInput{LinkColumn: linkColumnInput}

	// optional
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

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%smeta", path)); ok {
		meta, diags := newMonitorV2LinkColumnMetaInput(fmt.Sprintf("%smeta.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		column.Meta = meta
	}

	return column, diags
}

func newMonitorV2LinkColumnMetaInput(path string, data *schema.ResourceData) (meta *gql.MonitorV2LinkColumnMetaInput, diags diag.Diagnostics) {
	// instantiation
	meta = &gql.MonitorV2LinkColumnMetaInput{}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%ssrc_fields", path)); ok {
		srcFields := make([]gql.MonitorV2ColumnPathInput, 0)
		for i := range data.Get(fmt.Sprintf("%ssrc_fields", path)).([]interface{}) {
			srcField, diags := newMonitorV2ColumnPathInput(fmt.Sprintf("%ssrc_fields.%d.", path, i), data)
			if diags.HasError() {
				return nil, diags
			}
			srcFields = append(srcFields, *srcField)
		}
		meta.SrcFields = srcFields
	}
	if _, ok := data.GetOk(fmt.Sprintf("%sdst_fields", path)); ok {
		dstFields := make([]string, 0)
		for i := range data.Get(fmt.Sprintf("%sdst_fields", path)).([]interface{}) {
			dstField := data.Get(fmt.Sprintf("%sdst_fields.%d", path, i)).(string)
			dstFields = append(dstFields, dstField)
		}
		meta.DstFields = dstFields
	}
	if _, ok := data.GetOk(fmt.Sprintf("%starget_dataset", path)); ok {
		v := data.Get(fmt.Sprintf("%starget_dataset", path))
		meta.TargetDataset = types.Int64Scalar(v.(int64)).Ptr()
	}

	return meta, diags
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

	//	NOTE: I rely on the fact that sizeof(int) == sizeof(int64) on modern systems
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
