package observe

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

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
		CustomizeDiff: resourceServiceAccountTokenCustomizeDiff,
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
				Type:             schema.TypeString,
				Required:         true,
				Description:      descriptions.Get("service_account_token", "schema", "expiration"),
				DiffSuppressFunc: suppressExpirationDiffWithinOneHour,
			},
		},
	}
}

// The API gives us a granularity of hours when setting the token lifetime,
func suppressExpirationDiffWithinOneHour(k, old, new string, d *schema.ResourceData) bool {
	if old == "" || new == "" {
		return false
	}
	oldTime, err := time.Parse(time.RFC3339, old)
	if err != nil {
		return false
	}
	newTime, err := time.Parse(time.RFC3339, new)
	if err != nil {
		return false
	}

	// Suppress diff if the difference is within one hour
	return newTime.Sub(oldTime).Abs() < time.Hour
}

func resourceServiceAccountTokenCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if diff.HasChange("expiration") {
		expiratonStr, _ := diff.Get("expiration").(string)
		expiration, err := time.Parse(time.RFC3339, expiratonStr)
		if err != nil {
			return fmt.Errorf("invalid expiration: %v", err)
		}
		// We check this in the customize diff rather than the validations, since we only
		// want to validate this when the expiration is modified. We don't want terraform
		// runs to fail after we reach the expiration date.
		if time.Until(expiration) < time.Hour {
			return fmt.Errorf("expiration must be at least 1 hour in the future")
		}
	}
	return nil
}

func serviceAccountTokenToResourceData(token *rest.ServiceAccountApiTokenResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	data.SetId(token.Id)
	setResourceData("label", token.Label)
	setResourceData("description", token.Description)
	setResourceData("disabled", token.Disabled)
	setResourceData("expiration", token.Expiration.Format("2006-01-02T15:04:05Z07:00"))

	// Secret is only returned on create, so we only set it if it's present
	if token.Secret != nil {
		setResourceData("secret", *token.Secret)
	}

	return diags
}

func serviceAccountTokenCreateRequestFromResourceData(data *schema.ResourceData) (req *rest.ServiceAccountApiTokenCreateRequest, diags diag.Diagnostics) {
	expirationStr := data.Get("expiration").(string)
	expiration, err := time.Parse(time.RFC3339, expirationStr)
	if err != nil {
		return nil, diag.Errorf("invalid expiration: %v", err)
	}
	lifetimeHours := int(math.Ceil(time.Until(expiration).Hours()))

	req = &rest.ServiceAccountApiTokenCreateRequest{
		Label:         data.Get("label").(string),
		Description:   data.Get("description").(string),
		LifetimeHours: lifetimeHours,
	}
	return req, diags
}

func serviceAccountTokenUpdateRequestFromResourceData(data *schema.ResourceData) (req *rest.ServiceAccountApiTokenUpdateRequest, diags diag.Diagnostics) {
	req = &rest.ServiceAccountApiTokenUpdateRequest{}

	if data.HasChange("label") {
		label := data.Get("label").(string)
		req.Label = &label
	}

	if data.HasChange("description") {
		description := data.Get("description").(string)
		req.Description = &description
	}

	if data.HasChange("expiration") {
		expirationStr := data.Get("expiration").(string)
		expiration, err := time.Parse(time.RFC3339, expirationStr)
		if err != nil {
			return nil, diag.Errorf("invalid expiration: %v", err)
		}
		lifetimeHours := int(math.Ceil(time.Until(expiration).Hours()))
		req.LifetimeHours = &lifetimeHours
	}

	if data.HasChange("disabled") {
		disabled := data.Get("disabled").(bool)
		req.Disabled = &disabled
	}

	return req, diags
}

func resourceServiceAccountTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	serviceAccountOid, _ := oid.NewOID(data.Get("service_account").(string))
	req, diags := serviceAccountTokenCreateRequestFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	token, err := client.CreateServiceAccountApiToken(ctx, serviceAccountOid.Id, req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create service account API token",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(token.Id)
	return append(diags, serviceAccountTokenToResourceData(token, data)...)
}

func resourceServiceAccountTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	serviceAccountOid, _ := oid.NewOID(data.Get("service_account").(string))
	token, err := client.GetServiceAccountApiToken(ctx, serviceAccountOid.Id, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// If the token has been deleted due to being expired, avoid clearing ID so
			// Terraform won't try to recreate it with a past timestamp.
			// Otherwise all terraform runs after a token expires will error.
			if expStr, ok := data.Get("expiration").(string); ok && expStr != "" {
				if exp, parseErr := time.Parse(time.RFC3339, expStr); parseErr == nil {
					if time.Now().After(exp) {
						// Keep state; add a warning so users know the token no longer exists remotely.
						return diag.Diagnostics{diag.Diagnostic{
							Severity: diag.Warning,
							Summary:  "Service account token has expired and no longer exists",
							Detail:   "The token has expired and is presumed deleted in Observe. Keeping it in state to avoid recreation with a past expiration.",
						}}
					}
				}
			}
			// Not expired (or can't determine): mark resource missing so Terraform can recreate.
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve service account API token: %s", err.Error())
	}

	return serviceAccountTokenToResourceData(token, data)
}

func resourceServiceAccountTokenUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	serviceAccountOid, _ := oid.NewOID(data.Get("service_account").(string))

	req, diags := serviceAccountTokenUpdateRequestFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	token, err := client.UpdateServiceAccountApiToken(ctx, serviceAccountOid.Id, data.Id(), req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to update service account API token",
			Detail:   err.Error(),
		})
		return diags
	}

	return append(diags, serviceAccountTokenToResourceData(token, data)...)
}

func resourceServiceAccountTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	serviceAccountOid, _ := oid.NewOID(data.Get("service_account").(string))
	err := client.DeleteServiceAccountApiToken(ctx, serviceAccountOid.Id, data.Id())
	if err != nil {
		return diag.Errorf("failed to delete service account API token: %s", err.Error())
	}

	return diags
}

func resourceServiceAccountTokenImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Expect import ID in the form "{service_account_id}:{token_id}"
	parts := strings.Split(data.Id(), ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid import id %q; expected format {service_account_id}:{token_id}", data.Id())
	}
	serviceAccountId, err := types.StringToUserIdScalar(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid service_account id %q: %v", parts[0], err)
	}
	tokenID := parts[1]

	if err := data.Set("service_account", oid.UserOid(serviceAccountId).String()); err != nil {
		return nil, err
	}
	data.SetId(tokenID)

	// If secret is provided via environment variable, set it
	if secret := os.Getenv("SECRET"); secret != "" {
		if err := data.Set("secret", secret); err != nil {
			return nil, err
		}
	}
	return []*schema.ResourceData{data}, nil
}
