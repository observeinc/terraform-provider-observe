package observe

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceWorkspace() *schema.Resource {

	return &schema.Resource{
		Description:   descriptions.Get("workspace", "description"),
		CreateContext: resourceWorkspaceCreate,
		UpdateContext: resourceWorkspaceUpdate,
		ReadContext:   resourceWorkspaceRead,
		DeleteContext: resourceWorkspaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("workspace", "schema", "name"),
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "id"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

func newWorkspaceConfig(data *schema.ResourceData) (input *gql.WorkspaceInput, diags diag.Diagnostics) {
	label := data.Get("name").(string)
	input = &gql.WorkspaceInput{
		Label: &label,
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

	data.SetId(result.Id)
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

	if err := data.Set("name", workspace.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("id", workspace.Id); err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set("oid", workspace.Oid().String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceWorkspaceDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	// TODO: Remove this despicable hack
	//       If we don't pause here, the workspace resource test fails sometimes
	//       while deleting the workspace at the end of the test case due to some
	//       concurrent access issue on Postgres.
	//       It's unclear what the cause is or how difficult it would be to fix
	//       server-side; for now, adding a one-second delay seems to be sufficient
	//       as a temporary client-side workaround.
	d, _ := time.ParseDuration("1s")
	time.Sleep(d)

	if err := client.DeleteWorkspace(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete workspace: %s", err.Error())
	}
	return diags
}
