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

func resourceMonitorV2MuteRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2MuteRuleCreate,
		ReadContext:   resourceMonitorV2MuteRuleRead,
		UpdateContext: resourceMonitorV2MuteRuleUpdate,
		DeleteContext: resourceMonitorV2MuteRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// needed as input to CreateMonitorV2MuteRule
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			// fields of MonitorV2MuteRuleInput
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "name"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "icon_url"),
			},
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeFolder),
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "folder"),
			},
			"managed_by": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(),
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "managed_by"),
			},
			"monitor_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeMonitorV2),
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "monitor_id"),
			},
			"schedule": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "schedule"),
				Elem:        monitorV2MuteRuleScheduleResource(),
			},
			"criteria": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "criteria"),
				Elem:        monitorV2ComparisonExpressionResource(),
			},
			// computed fields
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"valid_from": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "valid_from"),
			},
			"valid_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "valid_to"),
			},
			"is_global": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "is_global"),
			},
			"is_conditional": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "is_conditional"),
			},
		},
	}
}

func monitorV2MuteRuleScheduleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2MuteScheduleTypes),
				DiffSuppressFunc: diffSuppressEnums,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.type"),
			},
			"one_time": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"schedule.0.one_time", "schedule.0.recurring"},
				Description:  descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.one_time"),
				Elem:         monitorV2OneTimeMuteScheduleResource(),
			},
			"recurring": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"schedule.0.one_time", "schedule.0.recurring"},
				Description:  descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.recurring"),
				Elem:         monitorV2RecurringMuteScheduleResource(),
			},
		},
	}
}

func monitorV2OneTimeMuteScheduleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"start_time": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateTimestamp,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.one_time.start_time"),
			},
			"end_time": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimestamp,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.one_time.end_time"),
			},
		},
	}
}

func monitorV2RecurringMuteScheduleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"cron_schedule": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.recurring.cron_schedule"),
				Elem:        monitorV2CronScheduleResource(),
			},
			"duration": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateTimeDuration,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.recurring.duration"),
			},
		},
	}
}

func monitorV2CronScheduleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"raw_cron": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.recurring.cron_schedule.raw_cron"),
			},
			"timezone": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateStringIsTimezone,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "schedule.recurring.cron_schedule.timezone"),
			},
		},
	}
}

func monitorV2ComparisonExpressionResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"operator": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2BooleanOperators),
				DiffSuppressFunc: diffSuppressEnums,
				Description:      descriptions.Get("monitor_v2_mute_rule", "schema", "criteria.operator"),
			},
			"compare_terms": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "criteria.compare_terms"),
				Elem:        monitorV2ComparisonTermResource(),
			},
		},
	}
}

func monitorV2ComparisonTermResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"column": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "criteria.compare_terms.column"),
				Elem:        monitorV2ColumnResource(),
			},
			"comparison": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "criteria.compare_terms.comparison"),
				Elem:        monitorV2ComparisonResource(),
			},
		},
	}
}

// CRUD operations

func resourceMonitorV2MuteRuleCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2MuteRuleInput(data)
	if diags.HasError() {
		return diags
	}

	workspaceID, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitorV2MuteRule(ctx, workspaceID.Id, input)
	if err != nil {
		return diag.Errorf("failed to create monitor v2 mute rule: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceMonitorV2MuteRuleRead(ctx, data, meta)...)
}

func resourceMonitorV2MuteRuleUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2MuteRuleInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2MuteRule(ctx, data.Id(), input)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2MuteRuleCreate(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to update monitor v2 mute rule: %s", err.Error())
	}

	return append(diags, resourceMonitorV2MuteRuleRead(ctx, data, meta)...)
}

func resourceMonitorV2MuteRuleDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorV2MuteRule(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor v2 mute rule: %s", err.Error())
	}
	return diags
}

func resourceMonitorV2MuteRuleRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	muteRule, err := client.GetMonitorV2MuteRule(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitor v2 mute rule: %s", err.Error())
	}

	if err := data.Set("workspace", oid.WorkspaceOid(muteRule.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", muteRule.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", muteRule.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if muteRule.Description != nil {
		if err := data.Set("description", *muteRule.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if muteRule.IconUrl != nil {
		if err := data.Set("icon_url", *muteRule.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("folder", oid.FolderOid(muteRule.FolderId, muteRule.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if muteRule.ManagedById != nil {
		if err := data.Set("managed_by", oid.OID{Id: *muteRule.ManagedById}.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if muteRule.MonitorID != nil {
		if err := data.Set("monitor_id", oid.MonitorV2Oid(*muteRule.MonitorID).String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("schedule", monitorV2FlattenMuteRuleSchedule(muteRule.Schedule)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if muteRule.Criteria != nil {
		if err := data.Set("criteria", []interface{}{monitorV2FlattenComparisonExpression(muteRule.Criteria)}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("valid_from", muteRule.ValidFrom.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if muteRule.ValidTo != nil {
		if err := data.Set("valid_to", muteRule.ValidTo.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("is_global", muteRule.IsGlobal); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("is_conditional", muteRule.IsConditional); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

// Helper functions for converting to GraphQL input

func newMonitorV2MuteRuleInput(data *schema.ResourceData) (input *gql.MonitorV2MuteRuleInput, diags diag.Diagnostics) {
	input = &gql.MonitorV2MuteRuleInput{
		Name: data.Get("name").(string),
	}

	// schedule (required)
	scheduleInput, diags := newMonitorV2MuteRuleScheduleInput("schedule.0.", data)
	if diags.HasError() {
		return nil, diags
	}
	input.Schedule = *scheduleInput

	// optionals
	if v, ok := data.GetOk("description"); ok {
		desc := v.(string)
		input.Description = &desc
	}

	if v, ok := data.GetOk("icon_url"); ok {
		iconUrl := v.(string)
		input.IconUrl = &iconUrl
	}

	if v, ok := data.GetOk("folder"); ok {
		folderOID, _ := oid.NewOID(v.(string))
		input.FolderId = &folderOID.Id
	}

	if v, ok := data.GetOk("managed_by"); ok {
		managedByOID, _ := oid.NewOID(v.(string))
		input.ManagedById = &managedByOID.Id
	}

	if v, ok := data.GetOk("monitor_id"); ok {
		monitorOID, _ := oid.NewOID(v.(string))
		input.MonitorID = &monitorOID.Id
	}

	if _, ok := data.GetOk("criteria"); ok {
		criteriaInput, diags := newMonitorV2ComparisonExpressionInput("criteria.0.", data)
		if diags.HasError() {
			return nil, diags
		}
		input.Criteria = criteriaInput
	}

	return input, diags
}

func newMonitorV2MuteRuleScheduleInput(path string, data *schema.ResourceData) (input *gql.MonitorV2MuteRuleScheduleInput, diags diag.Diagnostics) {
	scheduleType := gql.MonitorV2MuteScheduleType(toCamel(data.Get(fmt.Sprintf("%stype", path)).(string)))

	input = &gql.MonitorV2MuteRuleScheduleInput{
		Type: scheduleType,
	}

	if _, ok := data.GetOk(fmt.Sprintf("%sone_time", path)); ok {
		if scheduleType != gql.MonitorV2MuteScheduleTypeOnetime {
			return nil, diag.Errorf("schedule type must be 'OneTime' when one_time is specified")
		}

		oneTimeInput, diags := newMonitorV2OneTimeMuteScheduleInput(fmt.Sprintf("%sone_time.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		input.OneTime = oneTimeInput
	}

	if _, ok := data.GetOk(fmt.Sprintf("%srecurring", path)); ok {
		if scheduleType != gql.MonitorV2MuteScheduleTypeRecurring {
			return nil, diag.Errorf("schedule type must be 'Recurring' when recurring is specified")
		}

		recurringInput, diags := newMonitorV2RecurringMuteScheduleInput(fmt.Sprintf("%srecurring.0.", path), data)
		if diags.HasError() {
			return nil, diags
		}
		input.Recurring = recurringInput
	}

	return input, diags
}

func newMonitorV2OneTimeMuteScheduleInput(path string, data *schema.ResourceData) (input *gql.MonitorV2OneTimeMuteScheduleInput, diags diag.Diagnostics) {
	startTimeStr := data.Get(fmt.Sprintf("%sstart_time", path)).(string)
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	startTimeScalar := types.TimeScalar(startTime)

	input = &gql.MonitorV2OneTimeMuteScheduleInput{
		StartTime: startTimeScalar,
	}

	if v, ok := data.GetOk(fmt.Sprintf("%send_time", path)); ok {
		endTimeStr := v.(string)
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		endTimeScalar := types.TimeScalar(endTime)
		input.EndTime = &endTimeScalar
	}

	return input, diags
}

func newMonitorV2RecurringMuteScheduleInput(path string, data *schema.ResourceData) (input *gql.MonitorV2MuteCronScheduleInput, diags diag.Diagnostics) {
	cronScheduleInput, diags := newMonitorV2CronScheduleInput(fmt.Sprintf("%scron_schedule.0.", path), data)
	if diags.HasError() {
		return nil, diags
	}

	durationStr := data.Get(fmt.Sprintf("%sduration", path)).(string)
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	input = &gql.MonitorV2MuteCronScheduleInput{
		CronSchedule: *cronScheduleInput,
		Duration:     types.DurationScalar(duration),
	}

	return input, diags
}

func newMonitorV2CronScheduleInput(path string, data *schema.ResourceData) (input *gql.MonitorV2CronScheduleInput, diags diag.Diagnostics) {
	timezone := data.Get(fmt.Sprintf("%stimezone", path)).(string)

	input = &gql.MonitorV2CronScheduleInput{
		Timezone: timezone,
	}

	if v, ok := data.GetOk(fmt.Sprintf("%sraw_cron", path)); ok {
		rawCron := v.(string)
		input.RawCron = &rawCron
	}

	return input, diags
}

// Helper functions for flattening GraphQL output

func monitorV2FlattenMuteRuleSchedule(schedule gql.MonitorV2MuteRuleSchedule) []interface{} {
	scheduleMap := map[string]interface{}{
		"type": toSnake(string(schedule.Type)),
	}

	if schedule.OneTime != nil {
		scheduleMap["one_time"] = monitorV2FlattenOneTimeMuteSchedule(*schedule.OneTime)
	}

	if schedule.Recurring != nil {
		scheduleMap["recurring"] = monitorV2FlattenRecurringMuteSchedule(*schedule.Recurring)
	}

	return []interface{}{scheduleMap}
}

func monitorV2FlattenOneTimeMuteSchedule(oneTime gql.MonitorV2OneTimeMuteSchedule) []interface{} {
	oneTimeMap := map[string]interface{}{
		"start_time": oneTime.StartTime.String(),
	}

	if oneTime.EndTime != nil {
		oneTimeMap["end_time"] = oneTime.EndTime.String()
	}

	return []interface{}{oneTimeMap}
}

func monitorV2FlattenRecurringMuteSchedule(recurring gql.MonitorV2MuteCronSchedule) []interface{} {
	dur := time.Duration(int64(recurring.Duration))

	recurringMap := map[string]interface{}{
		"cron_schedule": monitorV2FlattenCronSchedule(recurring.CronSchedule),
		"duration":      dur.String(),
	}

	return []interface{}{recurringMap}
}

func monitorV2FlattenCronSchedule(cronSchedule gql.MonitorV2CronSchedule) []interface{} {
	cronMap := map[string]interface{}{
		"timezone": cronSchedule.Timezone,
	}

	if cronSchedule.RawCron != nil {
		cronMap["raw_cron"] = *cronSchedule.RawCron
	}

	return []interface{}{cronMap}
}
