package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceDefaultDashboardForTag() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("default_dashboard_for_tag", "description_data_source"),

		ReadContext: dataSourceDefaultDashboardForTagRead,

		Schema: map[string]*schema.Schema{
			"tag": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      descriptions.Get("default_dashboard_for_tag", "schema", "tag_data_source"),
			},
			"dashboard": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("default_dashboard_for_tag", "schema", "dashboard"),
			},
		},
	}
}

func dataSourceDefaultDashboardForTagRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	tag := data.Get("tag").(string)

	dashid, err := client.GetDefaultDashboardForTag(ctx, tag)
	if err != nil {
		return diag.FromErr(err)
	}
	data.SetId(tag)
	return defaultDashboardForTagToResourceData(tag, dashid, data)
}
