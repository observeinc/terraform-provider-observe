package observe

import (
	"context"
	"fmt"

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

func dataSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
	)

	defer func() {
		// right now SDK does not report where this error happened,
		// so we need to provide a little extra context
		for i := range diags {
			diags[i].Detail = fmt.Sprintf("Failed to read dataset %q", name)
		}
		return
	}()

	oid, _ := observe.NewOID(data.Get("workspace").(string))

	d, err := client.LookupDataset(oid.ID, name)

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(d.ID)

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = diag.FromErr(err)
		return
	}
	return nil
}
