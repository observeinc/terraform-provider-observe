package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

const (
	schemaRbacGroupIdDescription          = "RbacGroup ID. Either `name` or `id` must be provided."
	schemaRbacGroupOIDDescription         = "The Observe ID for rbacGroup."
	schemaRbacGroupNameDescription        = "RbacGroup Name. Either `name` or `id` must be provided."
	schemaRbacGroupDescriptionDescription = "RbacGroup description."
)

func dataSourceRbacGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe RbacGroup.",
		ReadContext: dataSourceRbacGroupRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaRbacGroupIdDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaRbacGroupNameDescription,
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupOIDDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupDescriptionDescription,
			},
			//TODO: other metadata
		},
	}
}

func dataSourceRbacGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	var r *gql.RbacGroup
	var err error

	if explicitId != "" {
		r, err = client.GetRbacGroup(ctx, explicitId)
	} else if name != "" {
		r, err = client.LookupRbacGroup(ctx, name)
	}

	// The Everyone group only exists if RBAC v2 is turned on for the customer.
	// However, we want to support creating RBAC v2 statements before it's turned on,
	// so we return the special Everyone group id 1 if name is "Everyone" and there
	// is no existing v1 group called "Everyone".
	if err != nil && name == "Everyone" {
		r = &gql.RbacGroup{
			Id:   fmt.Sprintf("o::%s:rbacgroup:%d", client.CustomerID, 1),
			Name: "Everyone",
		}
		err = nil
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	return rbacGroupToResourceData(r, data)
}

func rbacGroupToResourceData(r *gql.RbacGroup, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", r.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("description", r.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("oid", r.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	data.SetId(r.Id)
	return diags
}
