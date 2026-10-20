package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// The API has no way to read a token back, so every attribute is either
// ForceNew input or computed once at create time.
func resourceInboundShareRegistrationToken() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("inbound_share_registration_token", "description"),
		CreateContext: resourceInboundShareRegistrationTokenCreate,
		ReadContext:   resourceInboundShareRegistrationTokenRead,
		DeleteContext: resourceInboundShareRegistrationTokenDelete,
		Schema: map[string]*schema.Schema{
			"provider_account": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: descriptions.Get("inbound_share_registration_token", "schema", "provider_account"),
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: descriptions.Get("inbound_share_registration_token", "schema", "token"),
			},
			"setup_sql": {
				Type:        schema.TypeList,
				Computed:    true,
				Sensitive:   true,
				Elem:        inboundShareSetupSqlStepSchema(),
				Description: descriptions.Get("inbound_share_registration_token", "schema", "setup_sql"),
			},
			"expires_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_registration_token", "schema", "expires_at"),
			},
		},
	}
}

func resourceInboundShareRegistrationTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	providerAccount := data.Get("provider_account").(string)

	token, err := client.CreateShareRegistrationToken(ctx, &rest.CreateRegistrationTokenRequest{
		ProviderAccount: providerAccount,
	})
	if err != nil {
		return inboundShareErrorDiags("failed to create inbound share registration token", err)
	}

	data.SetId(providerAccount)
	values := map[string]interface{}{
		"token":      token.Token,
		"setup_sql":  flattenSetupSqlSteps(token.SetupSql),
		"expires_at": token.ExpiresAt,
	}
	for key, value := range values {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	return diags
}

func resourceInboundShareRegistrationTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceInboundShareRegistrationTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}
