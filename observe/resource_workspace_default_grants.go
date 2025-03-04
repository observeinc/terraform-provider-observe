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

type WorkspaceDefaultGrantPermission string

const (
	WorkspaceDefaultGrantPermissionView WorkspaceDefaultGrantPermission = "View"
	WorkspaceDefaultGrantPermissionEdit WorkspaceDefaultGrantPermission = "Edit"
)

var validWorkspaceDefaultGrantPermissions = []WorkspaceDefaultGrantPermission{
	WorkspaceDefaultGrantPermissionView,
	WorkspaceDefaultGrantPermissionEdit,
}

func resourceWorkspaceDefaultGrants() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("workspace_default_grants", "description"),
		CreateContext: resourceWorkspaceDefaultGrantsCreate,
		ReadContext:   resourceWorkspaceDefaultGrantsRead,
		UpdateContext: resourceWorkspaceDefaultGrantsUpdate,
		DeleteContext: resourceWorkspaceDefaultGrantsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     resourceWorkspaceDefaultGrantsGroup(),
			},
		},
	}
}

func resourceWorkspaceDefaultGrantsGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("workspace_default_grants", "schema", "group", "oid"),
			},
			"permission": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(validWorkspaceDefaultGrantPermissions),
				Description:      descriptions.Get("workspace_default_grants", "schema", "group", "permission"),
			},
		},
	}
}

func resourceWorkspaceDefaultGrantsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newWorkspaceDefaultGrantsInput(data)
	if diags.HasError() {
		return diags
	}

	if err := client.SetRbacDefaultSharingGroups(ctx, input); err != nil {
		return diag.Errorf("failed to set workspace default grants: %s", err.Error())
	}

	// we just set a constant id since there's only one of this config per tenant
	data.SetId("workspace_default_grants")
	return nil
}

func resourceWorkspaceDefaultGrantsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	groups, err := client.GetRbacDefaultSharingGroups(ctx)
	if err != nil {
		return diag.Errorf("failed to get workspace default grants: %s", err.Error())
	}

	return workspaceDefaultGrantsToResourceData(groups, data)
}

func resourceWorkspaceDefaultGrantsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return resourceWorkspaceDefaultGrantsCreate(ctx, data, meta)
}

func resourceWorkspaceDefaultGrantsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.SetRbacDefaultSharingGroups(ctx, []gql.RbacDefaultSharingGroupInput{}); err != nil {
		return diag.Errorf("failed to delete workspace default grants: %s", err.Error())
	}
	return diags
}

func workspaceDefaultGrantsToResourceData(sharingGroups []gql.RbacDefaultSharingGroup, data *schema.ResourceData) (diags diag.Diagnostics) {
	groups := schema.NewSet(schema.HashResource(resourceWorkspaceDefaultGrantsGroup()), []interface{}{})
	for _, g := range sharingGroups {
		permission := WorkspaceDefaultGrantPermissionView
		if g.AllowEdit {
			permission = WorkspaceDefaultGrantPermissionEdit
		}
		groups.Add(map[string]interface{}{
			"oid":        oid.RbacGroupOid(g.GroupId).String(),
			"permission": toSnake(string(permission)),
		})
	}
	if err := data.Set("group", groups); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func newWorkspaceDefaultGrantsInput(data *schema.ResourceData) (input []gql.RbacDefaultSharingGroupInput, diags diag.Diagnostics) {
	groups := data.Get("group").(*schema.Set).List()
	input = make([]gql.RbacDefaultSharingGroupInput, 0, len(groups))
	for _, g := range groups {
		group := g.(map[string]interface{})
		groupOid, err := oid.NewOID(group["oid"].(string))
		if err != nil {
			return nil, diag.Errorf("error parsing group oid: %s", err.Error())
		}
		permission := WorkspaceDefaultGrantPermission(toCamel(group["permission"].(string)))
		input = append(input, gql.RbacDefaultSharingGroupInput{
			GroupId:   groupOid.Id,
			AllowEdit: permission == WorkspaceDefaultGrantPermissionEdit,
		})
	}
	return input, diags
}
