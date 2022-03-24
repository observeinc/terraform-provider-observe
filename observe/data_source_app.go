package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceApp() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAppRead,
		Schema: map[string]*schema.Schema{
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(observe.TypeFolder),
				Description:      schemaDatasetWorkspaceDescription,
			},
			"module_id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"module_id", "id"},
				Optional:     true,
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"module_id", "id"},
				Optional:     true,
			},
			// computed values
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"variables": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAppRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client    = meta.(*observe.Client)
		module_id = data.Get("module_id").(string)
		id        = data.Get("id").(string)
	)

	oid, _ := observe.NewOID(data.Get("folder").(string))

	var m *observe.App
	var err error

	if id != "" {
		m, err = client.GetApp(ctx, id)
	} else if module_id != "" {
		m, err = client.LookupApp(ctx, oid.ID, module_id)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(m.ID)
	return resourceAppRead(ctx, data, meta)
}
