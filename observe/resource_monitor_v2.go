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
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// TODO: make the schema keys constants?
// annoying to change varnames in 3 non-obvious places

func resourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("monitorv2", "description"),
		CreateContext: resourceMonitorV2Create,
		ReadContext:   resourceMonitorV2Read,
		UpdateContext: resourceMonitorV2Update,
		DeleteContext: resourceMonitorV2Delete,
		Schema: map[string]*schema.Schema{
			"disabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "disabled"),
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "comment"),
			},
			"rule_kind": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "rule_kind"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitorv2", "schema", "name"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "icon_url"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "description"),
			},
			"managed_by_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "managed_by_id"),
			},
			"folder_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitorv2", "schema", "folder_id"),
			},
			"stage": {
				// for building the queries
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
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
			"rules": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"level": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitorv2", "schema", "rules", "level"),
						},
						"count": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_fn": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("monitorv2", "schema", "rules", "count", "compare_fn"),
									},
									"compare_value": {
										Type:        schema.TypeFloat,
										Required:    true,
										Description: descriptions.Get("monitorv2", "schema", "rules", "count", "compare_value"),
									},
								},
							},
						},
					},
				},
			},
			"lookback_time": {
				// TODO: this is supposed to be a duration, but it's a string
				// how to deal with this?
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("monitorv2", "schema", "lookback_time"),
			},
			"group_by_groups": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"columns": {
							Type:        schema.TypeList,
							Required:    true,
							Description: descriptions.Get("monitorv2", "schema", "group_by_groups", "columns"),
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"group_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitorv2", "schema", "group_by_groups", "group_name"),
						},
						"column_path": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"column": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("monitorv2", "schema", "group_by_groups", "column_path", "column"),
									},
									"path": {
										Type:        schema.TypeList,
										Required:    true,
										Description: descriptions.Get("monitorv2", "schema", "group_by_groups", "column_path", "path"),
									},
								},
							},
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
