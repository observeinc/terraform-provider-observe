package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceWorksheet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorksheetRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
				Description:      schemaWorksheetWorkspaceDescription,
			},
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Worksheet ID.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetOIDDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetNameDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetIconDescription,
			},
			"queries": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetJSONDescription,
			},
		},
	}
}

func dataSourceWorksheetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		id     = data.Get("id").(string)
	)

	ws, err := client.GetWorksheet(ctx, id)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(ws.ID)

	return worksheetToResourceData(ws, data)
}
