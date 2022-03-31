package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceWorkspace() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorkspaceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
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

func dataSourceWorkspaceRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client    = meta.(*observe.Client)
		name      = data.Get("name").(string)
		id        = data.Get("id").(string)
		workspace *observe.Workspace
		err       error
	)

	if name != "" {
		workspace, err = client.LookupWorkspace(ctx, name)
		if err != nil {
			err = fmt.Errorf("failed to retrieve workspace %q: %w", name, err)
			return diag.FromErr(err)
		}
	} else {
		workspace, err = client.GetWorkspace(ctx, id)
		if err != nil {
			err = fmt.Errorf("failed to retrieve workspace %q: %w", name, err)
			return diag.FromErr(err)
		}
	}

	if err := data.Set("name", workspace.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("id", workspace.ID); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", workspace.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("datasets", workspace.Datasets); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	data.SetId(workspace.ID)
	return nil
}
