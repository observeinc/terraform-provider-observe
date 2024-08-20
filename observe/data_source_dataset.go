package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceDataset() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe dataset.",
		ReadContext: dataSourceDatasetRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"name": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				RequiredWith:     []string{"workspace"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description: descriptions.Get("dataset", "schema", "name") +
					"One of `name` or `id` must be set. If `name` is provided, `workspace` must be set.",
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				ValidateDiagFunc: validateID(),
				Description: descriptions.Get("common", "schema", "id") +
					"One of `name` or `id` must be set.",
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
				Description: descriptions.Get("dataset", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"acceleration_disabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"path_cost": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "path_cost"),
			},
			"on_demand_materialization_length": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "on_demand_materialization_length"),
			},
			"freshness": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "freshness"),
			},
			"inputs": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "inputs"),
			},
			"stage": {
				Type:     schema.TypeList,
				Computed: true,
				// we need to declare optional, otherwise we won't get block
				// formatting in state
				Optional:    true,
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"output_stage": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
		},
	}
}

func dataSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	var d *gql.Dataset
	var err error

	if explicitId != "" {
		d, err = client.GetDataset(ctx, explicitId)
	} else {
		defer func() {
			// right now SDK does not report where this error happened,
			// so we need to provide a little extra context
			for i := range diags {
				diags[i].Detail = fmt.Sprintf("Failed to read dataset %q", name)
			}
		}()

		var implicitId *oid.OID
		implicitId, err = oid.NewOID(data.Get("workspace").(string))
		if err == nil {
			d, err = client.LookupDataset(ctx, implicitId.Id, name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(d.Id)

	return datasetToResourceData(d, data)
}
