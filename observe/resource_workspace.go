package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceWorkspace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceWorkspaceCreate,
		UpdateContext: resourceWorkspaceUpdate,
		ReadContext:   resourceWorkspaceRead,
		DeleteContext: resourceWorkspaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datasets": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func newWorkspaceConfig(data *schema.ResourceData) (config *observe.WorkspaceConfig, diags diag.Diagnostics) {
	config = &observe.WorkspaceConfig{
		Name: data.Get("name").(string),
	}
	return
}

func resourceWorkspaceCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newWorkspaceConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateWorkspace(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create workspace: %s", err.Error())
	}

	data.SetId(result.ID)
	return append(diags, resourceWorkspaceRead(ctx, data, meta)...)
}

func resourceWorkspaceUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newWorkspaceConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateWorkspace(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update workspace: %s", err.Error())
	}

	return append(diags, resourceWorkspaceRead(ctx, data, meta)...)
}

func resourceWorkspaceRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	workspace, err := client.GetWorkspace(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read workspace: %s", err.Error())
	}

	if err := data.Set("name", workspace.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", workspace.OID().String()); err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set("datasets", workspace.Datasets); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceWorkspaceDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteWorkspace(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete action: %s", err.Error())
	}
	return diags
}
