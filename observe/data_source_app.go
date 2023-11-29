package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceApp() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches data for an installed Observe app.",
		ReadContext: dataSourceAppRead,
		Schema: map[string]*schema.Schema{
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeFolder),
				Description:      schemaDatasetWorkspaceDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				RequiredWith: []string{"folder"},
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				ValidateDiagFunc: validateID,
				Optional:         true,
			},
			// computed values
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"module_id": {
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
			"outputs": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAppRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
		id     = data.Get("id").(string)
	)

	folderId, _ := oid.NewOID(data.Get("folder").(string))

	var m *gql.App
	var err error

	if id != "" {
		m, err = client.GetApp(ctx, id)
	} else if name != "" {
		m, err = client.LookupApp(ctx, folderId.Id, name)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(m.Id)
	return resourceAppRead(ctx, data, meta)
}
