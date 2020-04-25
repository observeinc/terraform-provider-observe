package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceWorkspace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceWorkspaceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceWorkspaceRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	var (
		name = d.Get("name").(string)
	)

	workspaces, err := c.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to retrieve workspaces: %w", err)
	}

	for _, w := range workspaces {
		if w.Label == name {
			d.SetId(w.ID)
			return nil
		}
	}

	return fmt.Errorf("workspace not found")
}
