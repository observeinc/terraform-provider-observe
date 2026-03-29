package observe

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("service_account_token", "description"),
		CreateContext: resourceServiceAccountTokenCreate,
		ReadContext:   resourceServiceAccountTokenRead,
		UpdateContext: resourceServiceAccountTokenUpdate,
		DeleteContext: resourceServiceAccountTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceServiceAccountTokenImport,
		},
		Schema: map[string]*schema.Schema{
			"service_account": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeUser),
				Description:      descriptions.Get("service_account_token", "schema", "service_account"),
			},
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("service_account_token", "schema", "label"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("service_account_token", "schema", "description"),
			},
			"lifetime_hours": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: descriptions.Get("service_account_token", "schema", "lifetime_hours"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("service_account_token", "schema", "disabled"),
			},
			"secret": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
				Description: descriptions.Get("service_account_token", "schema", "secret"),
			},
			"expiration": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("service_account_token", "schema", "expiration"),
			},
		},
	}
}

func serviceAccountTokenToResourceData(token *rest.ServiceAccountTokenResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	set := func(k string, v interface{}) {
		if err := data.Set(k, v); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	data.SetId(token.Id)
	set("label", token.Label)
	set("description", token.Description)
	set("disabled", token.Disabled)
	set("expiration", token.Expiration)
	if token.Secret != nil { // Secret is only returned on create
		set("secret", *token.Secret)
	}
	return diags
}

func resourceServiceAccountTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	accountOid, err := oid.NewOID(data.Get("service_account").(string))
	if err != nil {
		return diag.Errorf("invalid service_account OID: %s", err.Error())
	}

	req := &rest.ServiceAccountTokenCreateRequest{
		Label:         data.Get("label").(string),
		Description:   data.Get("description").(string),
		LifetimeHours: data.Get("lifetime_hours").(int),
	}
	token, err := client.Rest.CreateServiceAccountToken(ctx, accountOid.Id, req)
	if err != nil {
		return diag.Errorf("failed to create service account token: %s", err.Error())
	}

	data.SetId(token.Id)
	return append(diags, serviceAccountTokenToResourceData(token, data)...)
}

func resourceServiceAccountTokenUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	accountOid, err := oid.NewOID(data.Get("service_account").(string))
	if err != nil {
		return diag.Errorf("invalid service_account OID: %s", err.Error())
	}

	req := &rest.ServiceAccountTokenUpdateRequest{}
	if data.HasChange("label") {
		v := data.Get("label").(string)
		req.Label = &v
	}
	if data.HasChange("description") {
		v := data.Get("description").(string)
		req.Description = &v
	}
	if data.HasChange("lifetime_hours") {
		v := data.Get("lifetime_hours").(int)
		req.LifetimeHours = &v
	}
	if data.HasChange("disabled") {
		v := data.Get("disabled").(bool)
		req.Disabled = &v
	}

	token, err := client.Rest.UpdateServiceAccountToken(ctx, accountOid.Id, data.Id(), req)
	if err != nil {
		return diag.Errorf("failed to update service account token: %s", err.Error())
	}

	return append(diags, serviceAccountTokenToResourceData(token, data)...)
}

func resourceServiceAccountTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	accountOid, err := oid.NewOID(data.Get("service_account").(string))
	if err != nil {
		return diag.Errorf("invalid service_account OID: %s", err.Error())
	}

	result, err := client.Rest.GetServiceAccountToken(ctx, accountOid.Id, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// Token has already been deleted. Two possibilites:
			//   1. Token has expired and been auto-deleted by Observe.
			//   2. Token was deleted manually.
			// In both cases, we probably don't want to recreate the token automatically (with a new lifetime),
			// and instead just let the user manually do so if desired. So we don't clear the ID here.
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("service account token (id=%s) not found; it may have expired and been auto-deleted by Observe. Terraform will not recreate it automatically.", data.Id()),
			})
			return diags
		}
		return diag.Errorf("failed to retrieve service account token: %s", err.Error())
	}

	return serviceAccountTokenToResourceData(result, data)
}

func resourceServiceAccountTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	accountOid, err := oid.NewOID(data.Get("service_account").(string))
	if err != nil {
		return diag.Errorf("invalid service_account OID: %s", err.Error())
	}

	if err := client.Rest.DeleteServiceAccountToken(ctx, accountOid.Id, data.Id()); err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// already gone
			return diags
		}
		return diag.Errorf("failed to delete service account token: %s", err.Error())
	}
	return diags
}

func resourceServiceAccountTokenImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Import ID format: <service_account_id>/<token_id>
	parts := strings.Split(data.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("import ID must be in format <service_account_id>/<token_id>, got: %s", data.Id())
	}

	serviceAccountId, err := types.StringToUserIdScalar(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid service account ID: %s", err.Error())
	}
	tokenId := parts[1]

	if err := data.Set("service_account", oid.UserOid(serviceAccountId).String()); err != nil {
		return nil, err
	}
	data.SetId(tokenId)

	// If SECRET environment variable is set, use it for the secret field
	if secret := os.Getenv("SECRET"); secret != "" {
		if err := data.Set("secret", secret); err != nil {
			return nil, err
		}
	}

	return []*schema.ResourceData{data}, nil
}
