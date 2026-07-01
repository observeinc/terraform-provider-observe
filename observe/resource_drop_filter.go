package observe

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceDropFilter() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("drop_filter", "description"),
		CreateContext: resourceIngestFilterCreate,
		ReadContext:   resourceIngestFilterRead,
		UpdateContext: resourceIngestFilterUpdate,
		DeleteContext: resourceIngestFilterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			// The source dataset of a drop filter is immutable server-side, so a
			// change requires recreating the resource. The dataset OID embeds a
			// version that is diff-suppressed, so compare ids directly here
			// rather than setting ForceNew on the schema (which would trip on the
			// "known after apply" version whenever the dataset is updated). This
			// mirrors resource_dataset_query_filter.
			if d.HasChange("source_dataset") {
				oldVal, newVal := d.GetChange("source_dataset")
				oldOid, oldErr := oid.NewOID(oldVal.(string))
				newOid, newErr := oid.NewOID(newVal.(string))
				if oldErr == nil && newErr == nil && oldOid.Id != newOid.Id {
					d.ForceNew("source_dataset")
				}
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				DiffSuppressFunc: diffSuppressWorkspace,
				Deprecated:       "workspace is no longer required and will be ignored. It may be removed in a future version.",
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"pipeline": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("drop_filter", "schema", "pipeline"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("drop_filter", "schema", "name"),
			},
			"drop_rate": {
				Type:        schema.TypeFloat,
				Required:    true,
				Description: descriptions.Get("drop_filter", "schema", "drop_rate"),
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: descriptions.Get("drop_filter", "schema", "enabled"),
			},
			"source_dataset": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      descriptions.Get("drop_filter", "schema", "source_dataset"),
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
		},
	}
}

func ingestFilterToResourceData(filter *rest.IngestFilterResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	data.SetId(filter.Id)
	setResourceData("oid", filter.Oid().String())
	setResourceData("name", filter.Label)
	setResourceData("pipeline", filter.Pipeline)
	setResourceData("drop_rate", filter.DropRate)
	setResourceData("enabled", filter.Enabled)
	setResourceData("source_dataset", oid.DatasetOid(filter.SourceDataset.Id).String())
	// workspace is intentionally not set: the REST API does not return a
	// workspace, and the attribute is deprecated and ignored by the server.

	return diags
}

func resourceIngestFilterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	sourceDatasetOid, err := oid.NewOID(data.Get("source_dataset").(string))
	if err != nil {
		return diag.Errorf("invalid source_dataset OID: %s", err.Error())
	}

	req := &rest.IngestFilterCreateRequest{
		Label:         data.Get("name").(string),
		Pipeline:      data.Get("pipeline").(string),
		DropRate:      data.Get("drop_rate").(float64),
		Enabled:       data.Get("enabled").(bool),
		SourceDataset: rest.DatasetRef{Id: sourceDatasetOid.Id},
	}

	filter, err := client.CreateIngestFilter(ctx, req)
	if err != nil {
		return diag.Errorf("failed to create drop filter: %s", err.Error())
	}

	data.SetId(filter.Id)

	return append(diags, resourceIngestFilterRead(ctx, data, meta)...)
}

func resourceIngestFilterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	filter, err := client.GetIngestFilter(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve drop filter [id=%s]: %s", data.Id(), err.Error())
	}

	return ingestFilterToResourceData(filter, data)
}

func resourceIngestFilterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	req := &rest.IngestFilterUpdateRequest{
		Label:    data.Get("name").(string),
		Pipeline: data.Get("pipeline").(string),
		DropRate: data.Get("drop_rate").(float64),
		Enabled:  data.Get("enabled").(bool),
	}

	if _, err := client.UpdateIngestFilter(ctx, data.Id(), req); err != nil {
		return diag.Errorf("failed to update drop filter [id=%s]: %s", data.Id(), err.Error())
	}

	return append(diags, resourceIngestFilterRead(ctx, data, meta)...)
}

func resourceIngestFilterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteIngestFilter(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete drop filter: %s", err.Error())
	}
	return diags
}
