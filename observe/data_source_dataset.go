package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceDataset() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDatasetRead,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
	)

	oid, _ := observe.NewOID(data.Get("workspace").(string))

	d, err := client.LookupDataset(oid.ID, name)
	if err != nil {
		return diag.FromErr(err)
	}
	data.SetId(d.ID)

	if err := data.Set("oid", d.OID().String()); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
