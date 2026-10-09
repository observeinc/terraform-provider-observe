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
	schemaRbacGroupmemberIdDescription          = "RbacGroupmember ID."
	schemaRbacGroupmemberOIDDescription         = "The Observe ID for rbacGroupmember."
	schemaRbacGroupmemberGroupDescription       = "OID of the RbacGroup this membership belongs to."
	schemaRbacGroupmemberDescriptionDescription = "RbacGroupmember description."
	schemaRbacGroupmemberMemberDescription      = "The member of the group."
	schemaRbacGroupmemberMemberUserDescription  = "OID of the user that is a member of the group."
	schemaRbacGroupmemberMemberGroupDescription = "OID of the group that is a member of the group."
)

func dataSourceRbacGroupmember() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe RbacGroupmember.",
		ReadContext: dataSourceRbacGroupmemberRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaRbacGroupmemberIdDescription,
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupmemberOIDDescription,
			},
			"group": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupmemberGroupDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupmemberDescriptionDescription,
				Deprecated:  "Descriptions for group memberships are no longer supported.",
			},
			"member": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: schemaRbacGroupmemberMemberDescription,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: schemaRbacGroupmemberMemberUserDescription,
						},
						"group": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: schemaRbacGroupmemberMemberGroupDescription,
							Deprecated:  "Groups may no longer be members of other groups.",
						},
					},
				},
			},
		},
	}
}

func dataSourceRbacGroupmemberRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		id     = data.Get("id").(string)
	)

	r, err := client.GetRbacGroupmember(ctx, id)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	return rbacGroupmemberDataToResourceData(r, data)
}

func rbacGroupmemberDataToResourceData(r *gql.RbacGroupmember, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("group", oid.RbacGroupOid(r.GroupId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("description", r.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("oid", r.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	member := make(map[string]interface{}, 0)
	if r.MemberUserId != nil {
		member["user"] = oid.UserOid(*r.MemberUserId).String()
	} else if r.MemberGroupId != nil {
		member["group"] = oid.RbacGroupOid(*r.MemberGroupId).String()
	}
	if err := data.Set("member", []interface{}{member}); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	data.SetId(r.Id)
	return diags
}
