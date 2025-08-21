package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("service_account", "data_source_description"),
		ReadContext: dataSourceServiceAccountRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("service_account", "schema", "id"),
			},
			// computed values
			"label": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("service_account", "schema", "label"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("service_account", "schema", "description"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("service_account", "schema", "disabled"),
			},
		},
	}
}

func dataSourceServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	id := data.Get("id").(string)
	serviceAccount, err := client.GetServiceAccount(ctx, id)
	if err != nil {
		return diag.Errorf("failed to retrieve service account: %s", err.Error())
	}

	return serviceAccountToResourceData(serviceAccount, data)
}
