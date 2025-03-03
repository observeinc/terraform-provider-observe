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

type DefaultSharingGroupPermission string

const (
	DefaultSharingGroupPermissionView DefaultSharingGroupPermission = "View"
	DefaultSharingGroupPermissionEdit DefaultSharingGroupPermission = "Edit"
)

var validDefaultSharingGroupPermissions = []DefaultSharingGroupPermission{
	DefaultSharingGroupPermissionView,
	DefaultSharingGroupPermissionEdit,
}

func resourceDefaultSharingGroups() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("default_sharing_groups", "description"),
		CreateContext: resourceDefaultSharingGroupsCreate,
		ReadContext:   resourceDefaultSharingGroupsRead,
		UpdateContext: resourceDefaultSharingGroupsUpdate,
		DeleteContext: resourceDefaultSharingGroupsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     resourceDefaultSharingGroupsGroup(),
			},
		},
	}
}

func resourceDefaultSharingGroupsGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("default_sharing_groups", "schema", "group", "oid"),
			},
			"permission": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(validDefaultSharingGroupPermissions),
				Description:      descriptions.Get("default_sharing_groups", "schema", "group", "permission"),
			},
		},
	}
}

func resourceDefaultSharingGroupsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newDefaultSharingGroupsInput(data)
	if diags.HasError() {
		return diags
	}

	if err := client.SetRbacDefaultSharingGroups(ctx, input); err != nil {
		return diag.Errorf("failed to set default sharing groups: %s", err.Error())
	}

	// we just set a constant id since there's only one of this config per tenant
	data.SetId("default_sharing_groups")
	return nil
}

func resourceDefaultSharingGroupsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	groups, err := client.GetRbacDefaultSharingGroups(ctx)
	if err != nil {
		return diag.Errorf("failed to get default sharing groups: %s", err.Error())
	}

	return defaultSharingGroupsToResourceData(groups, data)
}

func resourceDefaultSharingGroupsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return resourceDefaultSharingGroupsCreate(ctx, data, meta)
}

func resourceDefaultSharingGroupsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.SetRbacDefaultSharingGroups(ctx, []gql.RbacDefaultSharingGroupInput{}); err != nil {
		return diag.Errorf("failed to delete default sharing groups: %s", err.Error())
	}
	return diags
}

func defaultSharingGroupsToResourceData(groups []gql.RbacDefaultSharingGroup, data *schema.ResourceData) (diags diag.Diagnostics) {
	// sharingGroups := make([]map[string]interface{}, 0)
	sharingGroups := schema.NewSet(schema.HashResource(resourceDefaultSharingGroupsGroup()), []interface{}{})
	for _, g := range groups {
		permission := DefaultSharingGroupPermissionView
		if g.AllowEdit {
			permission = DefaultSharingGroupPermissionEdit
		}
		sharingGroups.Add(map[string]interface{}{
			"oid":        oid.RbacGroupOid(g.GroupId).String(),
			"permission": toSnake(string(permission)),
		})
	}
	if err := data.Set("group", sharingGroups); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func newDefaultSharingGroupsInput(data *schema.ResourceData) (input []gql.RbacDefaultSharingGroupInput, diags diag.Diagnostics) {
	groups := data.Get("group").(*schema.Set).List()
	input = make([]gql.RbacDefaultSharingGroupInput, 0, len(groups))
	for _, g := range groups {
		group := g.(map[string]interface{})
		groupOid, err := oid.NewOID(group["oid"].(string))
		if err != nil {
			return nil, diag.Errorf("error parsing group oid: %s", err.Error())
		}
		permission := DefaultSharingGroupPermission(toCamel(group["permission"].(string)))
		input = append(input, gql.RbacDefaultSharingGroupInput{
			GroupId:   groupOid.Id,
			AllowEdit: permission == DefaultSharingGroupPermissionEdit,
		})
	}
	return input, diags
}
