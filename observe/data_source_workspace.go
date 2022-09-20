package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceWorkspace() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("workspace", "description"),
		ReadContext: dataSourceWorkspaceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description: descriptions.Get("workspace", "schema", "name") +
					"One of `id` or `name` must be provided",
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description: descriptions.Get("common", "schema", "id") +
					"One of `id` or `name` must be provided",
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
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
		workspace *gql.Workspace
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

	if err := data.Set("name", workspace.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("id", workspace.Id); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", workspace.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	datasets := make(map[string]string)
	for _, ds := range workspace.Datasets {
		datasets[ds.Label] = ds.Id
	}
	if err := data.Set("datasets", datasets); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	data.SetId(workspace.Id)
	return nil
}
