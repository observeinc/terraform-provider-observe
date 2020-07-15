package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceWorkspace() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorkspaceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datasets": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceWorkspaceRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		observe = meta.(*client.Client)
		name    = data.Get("name").(string)
	)

	workspace, err := observe.LookupWorkspace(name)
	if err != nil {
		err = fmt.Errorf("failed to retrieve workspace %q: %w", name, err)
		return diag.FromErr(err)
	}

	if err := data.Set("oid", workspace.OID().String()); err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set("datasets", workspace.Datasets); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(workspace.ID)
	return nil
}
