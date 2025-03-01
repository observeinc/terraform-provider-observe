package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	schemaRbacDefaultGroupDescription = "The Observe ID for rbacGroup. Currently API only accepts the ID of `reader`, `writer` or `admin`."
)

func resourceRbacDefaultGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Deprecated. Only for use with RBAC v1.\n\nManages a RBAC default group.",
		DeprecationMessage: "RBAC Default Groups are deprecated and only for use with RBAC v1." +
			" For v2, configure permissions on the Everyone group instead, which always includes all users." +
			" See `observe_rbac_group` and `observe_grant`.",
		CreateContext: resourceRbacDefaultGroupSet,
		UpdateContext: resourceRbacDefaultGroupSet,
		ReadContext:   resourceRbacDefaultGroupRead,
		DeleteContext: resourceRbacDefaultGroupUnset,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      schemaRbacDefaultGroupDescription,
				ValidateDiagFunc: validateOID(oid.TypeRbacGroup),
			},
		},
	}
}

func resourceRbacDefaultGroupSet(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	group, _ := oid.NewOID(data.Get("group").(string))

	if err := client.SetRbacDefaultGroup(ctx, group.Id); err != nil {
		return diag.Errorf("failed to set rbac default group: %s", err.Error())
	}
	data.SetId(group.Id)
	return append(diags, resourceRbacDefaultGroupRead(ctx, data, meta)...)
}

func resourceRbacDefaultGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	group, err := client.GetRbacDefaultGroup(ctx)
	if err != nil {
		return diag.Errorf("failed to read rbac default group: %s", err.Error())
	}
	return rbacDefaultGroupToResourceData(group, data)
}

func resourceRbacDefaultGroupUnset(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	if err := client.UnsetRbacDefaultGroup(ctx); err != nil {
		return diag.Errorf("failed to unset rbac default group: %s", err.Error())
	}
	return diags
}

func rbacDefaultGroupToResourceData(r *gql.RbacGroup, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("group", r.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}
