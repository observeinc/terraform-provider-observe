package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

const (
	schemaUserIdDescription      = "User ID. Either `email` or `id` must be provided."
	schemaUserOIDDescription     = "The Observe ID for user."
	schemaUserEmailDescription   = "User Email. Either `email` or `id` must be provided."
	schemaUserCommentDescription = "User comment."
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe user.",
		ReadContext: dataSourceUserRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"email", "id"},
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateUID(),
				Description:      schemaUserIdDescription,
			},
			"email": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"email", "id"},
				Optional:     true,
				Computed:     true,
				Description:  schemaUserEmailDescription,
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaUserOIDDescription,
			},
			"comment": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaUserCommentDescription,
			},
		},
	}
}

func dataSourceUserRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		email      = data.Get("email").(string)
		explicitId = data.Get("id").(string)
	)

	var u *gql.User
	var err error

	if explicitId != "" {
		u, err = client.GetUser(ctx, explicitId)
	} else {
		u, err = client.LookupUser(ctx, email)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	return userToResourceData(u, data)
}

func userToResourceData(u *gql.User, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("email", u.Email); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if u.Comment != nil {
		if err := data.Set("comment", u.Comment); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if err := data.Set("oid", u.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	data.SetId(u.Id.String())
	return diags
}
