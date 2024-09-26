package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceReferenceTable() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe reference table.",
		ReadContext: dataSourceReferenceTableRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaReferenceTableWorkspaceDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaReferenceTableNameDescription,
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				ValidateDiagFunc: validateID(),
				Description:      "Reference table ID. Either `name` or `id` must be provided.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaReferenceTableOIDDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaReferenceTableDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaReferenceTableIconDescription,
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaReferenceTableDatasetDescription,
			},
		},
	}
}

func dataSourceReferenceTableRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
		getID  = data.Get("id").(string)
	)

	var m *gql.ReferenceTable
	var err error

	if getID != "" {
		m, err = client.GetReferenceTable(ctx, getID)
	} else if name != "" {
		workspaceID, _ := data.Get("workspace").(string)
		if workspaceID != "" {
			m, err = client.LookupReferenceTable(ctx, &workspaceID, &name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	} else if m == nil {
		return diag.Errorf("failed to lookup reference table from provided get/search parameters")
	}

	data.SetId(m.Id)
	return resourceReferenceTableRead(ctx, data, meta)
}
