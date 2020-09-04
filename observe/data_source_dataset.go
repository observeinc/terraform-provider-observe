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
			// computed values
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path_cost": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"freshness": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"inputs": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"stage": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				// we need to declare optional, otherwise we won't get block
				// formatting in state
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"input": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"pipeline": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
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

	return datasetToResourceData(d, data)
}
