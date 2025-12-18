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

func dataSourceMonitorV2MuteRule() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMonitorV2MuteRuleRead,
		Schema: map[string]*schema.Schema{
			// lookup parameters
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				ExactlyOneOf:     []string{"id", "workspace"},
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"id": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateID(),
				ExactlyOneOf:     []string{"id", "workspace"},
				Description:      descriptions.Get("common", "schema", "id"),
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"workspace"},
				Description:  descriptions.Get("monitor_v2_mute_rule", "schema", "name"),
			},
			// fields of MonitorV2MuteRule
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "icon_url"),
			},
			"folder": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "folder"),
			},
			"managed_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "managed_by"),
			},
			"monitor_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "monitor_id"),
			},
			"schedule": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_mute_rule", "schema", "schedule"),
				Elem:        monitorV2MuteRuleScheduleResource(),
			},
			"criteria": {
				Type:        schema.TypeList,
				Computed:    true,
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

func dataSourceMonitorV2MuteRuleRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var muteRule *gql.MonitorV2MuteRule
	client := meta.(*observe.Client)

	// lookup by id or by workspace+name
	if v, ok := data.GetOk("id"); ok {
		id := v.(string)
		var err error
		muteRule, err = client.GetMonitorV2MuteRule(ctx, id)
		if err != nil {
			return diag.Errorf("failed to read monitor v2 mute rule: %s", err.Error())
		}
	} else {
		workspaceOID, err := oid.NewOID(data.Get("workspace").(string))
		if err != nil {
			return diag.FromErr(err)
		}
		var nameExact *string
		if v, ok := data.GetOk("name"); ok {
			name := v.(string)
			nameExact = &name
		}

		results, err := client.SearchMonitorV2MuteRule(ctx, &workspaceOID.Id, nameExact)
		if err != nil {
			return diag.Errorf("failed to search monitor v2 mute rule: %s", err.Error())
		}

		if len(results) == 0 {
			return diag.Errorf("no monitor v2 mute rule found matching criteria")
		}
		if len(results) > 1 {
			return diag.Errorf("found multiple monitor v2 mute rules matching criteria")
		}

		muteRule = &results[0]
	}

	data.SetId(muteRule.Id)

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
