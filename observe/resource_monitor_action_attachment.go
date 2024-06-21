package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceMonitorActionAttachment() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("monitor_action_attachment", "description"),
		CreateContext: resourceMonitorActionAttachmentCreate,
		ReadContext:   resourceMonitorActionAttachmentRead,
		UpdateContext: resourceMonitorActionAttachmentUpdate,
		DeleteContext: resourceMonitorActionAttachmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("monitor_action_attachment", "schema", "name"),
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
				Description: descriptions.Get("monitor_action_attachment", "schema", "description"),
			},
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"monitor": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeMonitor),
				Description:      descriptions.Get("monitor_action_attachment", "schema", "monitor"),
			},
			"action": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeMonitorAction),
				Description:      descriptions.Get("monitor_action_attachment", "schema", "action"),
			},
		},
	}
}

func newMonitorActionAttachmentConfig(data *schema.ResourceData) (input *gql.MonitorActionAttachmentInput, diags diag.Diagnostics) {
	id, err := oid.NewOID(data.Get("workspace").(string))
	if err != nil {
		return nil, diag.Errorf("failed to get monitor action workspace id: %s", err.Error())
	}
	input = &gql.MonitorActionAttachmentInput{
		WorkspaceId: id.Id,
		MonitorID:   data.Get("monitor").(string),
		ActionID:    data.Get("action").(string),
	}

	if v, ok := data.GetOk("name"); ok {
		input.Name = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return
}

func resourceMonitorActionAttachmentCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorActionAttachmentConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateMonitorActionAttachment(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId((*result).GetId())
	return append(diags, resourceMonitorActionAttachmentRead(ctx, data, meta)...)
}

func resourceMonitorActionAttachmentUpdate(ctx context.Context, data *schema.ResourceData, metaClient interface{}) (diags diag.Diagnostics) {
	client := metaClient.(*observe.Client)

	config, diags := newMonitorActionAttachmentConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorActionAttachment(ctx, data.Id(), config)
	if err != nil {
		if meta.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorActionAttachmentCreate(ctx, data, metaClient)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to update monitor action: %s", err.Error())
	}

	return append(diags, resourceMonitorActionAttachmentRead(ctx, data, metaClient)...)
}

func resourceMonitorActionAttachmentRead(ctx context.Context, data *schema.ResourceData, metaClient interface{}) (diags diag.Diagnostics) {
	client := metaClient.(*observe.Client)

	monitorActionAttachmentPtr, err := client.GetMonitorActionAttachment(ctx, data.Id())
	if err != nil {
		if meta.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitor action: %s", err.Error())
	}

	if err := data.Set("workspace", oid.WorkspaceOid(monitorActionAttachmentPtr.GetWorkspaceId()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", monitorActionAttachmentPtr.GetName()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("monitor", oid.MonitorOid(monitorActionAttachmentPtr.GetMonitorID()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("action", oid.MonitorActionOid(monitorActionAttachmentPtr.GetActionID()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", monitorActionAttachmentPtr.GetIconUrl()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitorActionAttachmentPtr.GetDescription()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", gql.MonitorActionAttachmentOid(*monitorActionAttachmentPtr).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourceMonitorActionAttachmentDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorActionAttachment(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor: %s", err.Error())
	}
	return diags
}
