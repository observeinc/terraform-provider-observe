package observe

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("service_account", "description"),
		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
		UpdateContext: resourceServiceAccountUpdate,
		DeleteContext: resourceServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// computed values
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("service_account", "schema", "id"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			// required values
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("service_account", "schema", "label"),
			},
			// optional values
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("service_account", "schema", "description"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("service_account", "schema", "disabled"),
			},
		},
	}
}

func serviceAccountToResourceData(serviceAccount *rest.ServiceAccountResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	data.SetId(serviceAccount.Id)
	setResourceData("oid", serviceAccount.Oid().String())
	setResourceData("label", serviceAccount.Label)
	setResourceData("description", serviceAccount.Description)
	setResourceData("disabled", serviceAccount.Disabled)

	return diags
}

func serviceAccountDefinitionFromResourceData(data *schema.ResourceData) (req *rest.ServiceAccountDefinition, diags diag.Diagnostics) {
	req = &rest.ServiceAccountDefinition{}

	req.Label = data.Get("label").(string)

	if v, ok := data.GetOk("description"); ok {
		req.Description = v.(string)
	}

	req.Disabled = data.Get("disabled").(bool)

	return req, diags
}

func resourceServiceAccountCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := serviceAccountDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	serviceAccount, err := client.CreateServiceAccount(ctx, req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create service account",
			Detail:   err.Error(),
		})
		return diags
	}
	data.SetId(serviceAccount.Id)

	return append(diags, serviceAccountToResourceData(serviceAccount, data)...)
}

func resourceServiceAccountUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := serviceAccountDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	serviceAccount, err := client.UpdateServiceAccount(ctx, data.Id(), req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to update service account",
			Detail:   err.Error(),
		})
		return diags
	}

	return append(diags, serviceAccountToResourceData(serviceAccount, data)...)
}

func resourceServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetServiceAccount(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve service account: %s", err.Error())
	}

	return serviceAccountToResourceData(result, data)
}

func resourceServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	err := client.DeleteServiceAccount(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to delete service account: %s", err.Error())
	}
	return diags
}
