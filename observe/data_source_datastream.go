package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceDatastream() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe datastream.",
		ReadContext: dataSourceDatastreamRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaDatastreamWorkspaceDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaDatastreamNameDescription,
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				ValidateDiagFunc: validateID,
				Description:      "Datastream ID. Either `name` or `id` must be provided.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamOIDDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamIconDescription,
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamDatasetDescription,
			},
		},
	}
}

func dataSourceDatastreamRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	implicitId, _ := oid.NewOID(data.Get("workspace").(string))

	var d *gql.Datastream
	var err error

	if explicitId != "" {
		d, err = client.GetDatastream(ctx, explicitId)
	} else if name != "" {
		defer func() {
			// right now SDK does not report where this error happened,
			// so we need to provide a little extra context
			for i := range diags {
				diags[i].Detail = fmt.Sprintf("Failed to read datastream %q", name)
			}
		}()

		d, err = client.LookupDatastream(ctx, implicitId.Id, name)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(d.Id)
	return datastreamToResourceData(d, data)
}
