package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceReferenceTable() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("reference_table", "description"),
		ReadContext: dataSourceReferenceTableRead,
		Schema: map[string]*schema.Schema{
			"label": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"label", "id"},
				Optional:     true,
				Description: descriptions.Get("reference_table", "schema", "label") +
					"One of `label` or `id` must be set.",
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"label", "id"},
				Optional:         true,
				ValidateDiagFunc: validateID(),
				Description: descriptions.Get("common", "schema", "id") +
					"One of `label` or `id` must be set.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("reference_table", "schema", "description"),
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("reference_table", "schema", "dataset"),
			},
			"checksum": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("reference_table", "schema", "checksum"),
			},
			"schema": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Description: descriptions.Get("reference_table", "schema", "schema"),
			},
			// TODO: add primary_key and label_field after API includes them in response
			// "primary_key": {
			// 	Type:        schema.TypeList,
			// 	Computed:    true,
			// 	Elem:        &schema.Schema{Type: schema.TypeString},
			// 	Description: descriptions.Get("reference_table", "schema", "primary_key"),
			// },
			// "label_field": {
			// 	Type:        schema.TypeString,
			// 	Computed:    true,
			// 	Description: descriptions.Get("reference_table", "schema", "label_field"),
			// },
		},
	}
}

func dataSourceReferenceTableRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	id := data.Get("id").(string)
	label := data.Get("label").(string)

	var m *rest.ReferenceTable
	var err error
	if id != "" {
		m, err = client.GetReferenceTable(ctx, id)
	} else if label != "" {
		m, err = client.LookupReferenceTable(ctx, label)
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
