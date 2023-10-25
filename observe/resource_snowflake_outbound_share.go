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

func resourceSnowflakeOutboundShare() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an outbound Snowflake share, which allows you to share datasets with an existing Snowflake account.",
		CreateContext: resourceSnowflakeOutboundShareCreate,
		ReadContext:   resourceSnowflakeOutboundShareRead,
		UpdateContext: resourceSnowflakeOutboundShareUpdate,
		DeleteContext: resourceSnowflakeOutboundShareDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A descriptive name for the share. This will be included in the Snowflake share name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the share.",
			},
			"account": {
				Required: true,
				MinItems: 1,
				Type:     schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"account": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the Snowflake account to share with.",
						},
						"organization": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the Snowflake organization to share with.",
						},
					},
				},
			},
			"share_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fully qualified name of the Snowflake share, including Observe's organization and account names.",
			},
		},
	}
}

func newSnowflakeOutboundShare(d *schema.ResourceData) (*gql.SnowflakeOutboundShareInput, diag.Diagnostics) {
	input := &gql.SnowflakeOutboundShareInput{
		Name:     d.Get("name").(string),
		Accounts: expandSnowflakeOutboundShareAccounts(d.Get("account").(*schema.Set).List()),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return input, nil
}

func expandSnowflakeOutboundShareAccounts(in []interface{}) []gql.SnowflakeAccountInput {
	out := make([]gql.SnowflakeAccountInput, 0)

	for _, v := range in {
		a := v.(map[string]interface{})
		out = append(out, gql.SnowflakeAccountInput{
			Account:      a["account"].(string),
			Organization: a["organization"].(string),
		})
	}

	return out

}

func resourceSnowflakeOutboundShareCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	id, err := oid.NewOID(d.Get("workspace").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	input, diags := newSnowflakeOutboundShare(d)
	if diags.HasError() {
		return diags
	}

	share, err := client.CreateSnowflakeOutboundShare(ctx, id.Id, input)
	if err != nil {
		return diag.Errorf("failed to create snowflake outbound share: %s", err)
	}

	d.SetId(share.Id)

	return append(diags, resourceSnowflakeOutboundShareRead(ctx, d, m)...)
}

func resourceSnowflakeOutboundShareRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	share, err := client.GetSnowflakeOutboundShare(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	if err := d.Set("workspace", oid.WorkspaceOid(share.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("oid", share.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("name", share.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if share.Description != nil {
		if err := d.Set("description", *share.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := d.Set("account", flattenSnowflakeOutboundShareAccounts(share.Accounts)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("share_name", share.ShareName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenSnowflakeOutboundShareAccounts(accounts []gql.SnowflakeAccount) []map[string]interface{} {
	var out []map[string]interface{}

	for _, account := range accounts {
		out = append(out, map[string]interface{}{
			"account":      account.Account,
			"organization": account.Organization,
		})
	}

	return out
}

func resourceSnowflakeOutboundShareUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	input, diags := newSnowflakeOutboundShare(d)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateSnowflakeOutboundShare(ctx, d.Id(), input)
	if err != nil {
		return diag.Errorf("failed to update snowflake outbound share: %s", err)
	}

	return append(diags, resourceSnowflakeOutboundShareRead(ctx, d, m)...)
}

func resourceSnowflakeOutboundShareDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	err := client.DeleteSnowflakeOutboundShare(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete snowflake outbound share: %s", err)
	}

	return nil
}
