package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceCloudInfo() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("cloud_info", "description"),
		ReadContext: dataSourceCloudInfoRead,
		Schema: map[string]*schema.Schema{
			// computed values
			"account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("cloud_info", "schema", "account_id"),
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("cloud_info", "schema", "region"),
			},
			"cloud_provider": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("cloud_info", "schema", "cloud_provider"),
			},
		},
	}
}

func dataSourceCloudInfoRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
	)

	info, err := client.GetCloudInfo(ctx)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	return cloudInfoToResourceData(info, data)
}

func cloudInfoToResourceData(info *gql.CloudInfo, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("account_id", info.AccountId); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("region", info.Region); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("cloud_provider", info.Provider); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	data.SetId(info.AccountId)
	return diags
}
