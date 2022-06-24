package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceDefaultDashboard() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDefaultDashboardRead,

		Schema: map[string]*schema.Schema{
			"dataset": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
			},
			// computed values
			"dashboard": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDefaultDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
	)

	dsid, _ := oid.NewOID(data.Get("dataset").(string))

	dashid, err := client.GetDefaultDashboard(ctx, dsid.Id)

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(dsid.Id)
	return defaultDashboardToResourceData(dsid.Id, dashid, data)
}
