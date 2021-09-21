package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceDatastream() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDatastreamRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
				Description:      schemaDatastreamWorkspaceDescription,
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  schemaDatastreamNameDescription,
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  "Datastream ID. Either `name` or `id` must be provided.",
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
		},
	}
}

func dataSourceDatastreamRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
		id     = data.Get("id").(string)
	)

	oid, _ := observe.NewOID(data.Get("workspace").(string))

	var d *observe.Datastream
	var err error

	if id != "" {
		d, err = client.GetDatastream(ctx, id)
	} else if name != "" {
		defer func() {
			// right now SDK does not report where this error happened,
			// so we need to provide a little extra context
			for i := range diags {
				diags[i].Detail = fmt.Sprintf("Failed to read datastream %q", name)
			}
			return
		}()

		d, err = client.LookupDatastream(ctx, oid.ID, name)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(d.ID)

	return datastreamToResourceData(d, data)
}
