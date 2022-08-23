package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceDashboard() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDashboardRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Dashboard ID.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardOIDDescription,
			},
			"workspace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardNameDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardIconDescription,
			},
			"stages": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardJSONDescription,
			},
			"layout": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardLayoutDescription,
			},
			"parameters": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardParametersDescription,
			},
			"parameter_values": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardParameterValuesDescription,
			},
		},
	}
}

func dataSourceDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		id     = data.Get("id").(string)
	)

	ws, err := client.GetDashboard(ctx, id)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(ws.Id)

	return dashboardToResourceData(ws, data)
}
