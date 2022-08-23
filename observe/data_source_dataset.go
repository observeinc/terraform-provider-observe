package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceDataset() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDatasetRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaDatasetWorkspaceDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaDatasetNameDescription,
				RequiredWith: []string{"workspace"},
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  "Dataset ID. Either `name` or `id` must be provided.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetOIDDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetIconDescription,
			},
			"path_cost": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: schemaDatasetPathCostDescription,
			},
			"freshness": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetFreshnessDescription,
			},
			"on_demand_materialization_length": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetOnDemandMaterializationLengthDescription,
			},
			"inputs": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: schemaDatasetInputsDescription,
			},
			"stage": {
				Type:     schema.TypeList,
				Computed: true,
				// we need to declare optional, otherwise we won't get block
				// formatting in state
				Optional:    true,
				Description: schemaDatasetStageDescription,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: schemaDatasetStageAliasDescription,
						},
						"input": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: schemaDatasetStageInputDescription,
						},
						"pipeline": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: schemaDatasetStagePipelineDescription,
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
	} else if name != "" {
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
