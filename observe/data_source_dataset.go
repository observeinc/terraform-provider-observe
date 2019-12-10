package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceDataset() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDatasetRead,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"label": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceDatasetRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	var (
		workspaceID = d.Get("workspace").(string)
		label       = d.Get("label").(string)
	)

	dataset, err := c.LookupDataset(workspaceID, label)
	if err != nil {
		return err
	}
	d.SetId(dataset.ID)
	return nil
}
