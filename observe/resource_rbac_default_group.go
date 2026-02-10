package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

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
	return diag.Errorf("observe_rbac_default_group is deprecated and no longer has any effect; use observe_workspace_default_grants / default sharing groups instead")
}

func resourceRbacDefaultGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	// We should error similar to the other operations, but to avoid breaking the terraform plans for
	// unrelated changes, we'll just no-op instead, leaving the existing state as is.
	// Only when someone attempts to change this resource will they get an error.
	return nil
}

func resourceRbacDefaultGroupUnset(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return diag.Errorf("observe_rbac_default_group is deprecated and no longer has any effect; use observe_workspace_default_grants / default sharing groups instead")
}
