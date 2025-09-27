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

func resourceDatasetQueryFilter() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("dataset_query_filter", "description"),
		CreateContext: resourceDatasetQueryFilterCreate,
		ReadContext:   resourceDatasetQueryFilterRead,
		UpdateContext: resourceDatasetQueryFilterUpdate,
		DeleteContext: resourceDatasetQueryFilterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"dataset": {
				Type:     schema.TypeString,
				Required: true,
				// We should be enforcing ForceNew, but unfortunately because the dataset oids contain a version,
				// the value of the oid is always "(known after apply)" when a dataset is updated. This means
				// that even though we're diff suppressing the version, the dataset query filter will always
				// be recreated when the dataset is updated, because terraform can't guarantee the new value of
				// the oid during the plan stage. So not setting ForceNew for now.
				// ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
				Description:      descriptions.Get("dataset_query_filter", "schema", "dataset"),
			},
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("dataset_query_filter", "schema", "label"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("dataset_query_filter", "schema", "description"),
			},
			"filter": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("dataset_query_filter", "schema", "filter"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("dataset_query_filter", "schema", "disabled"),
			},
			// Computed attributes
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"errors": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: descriptions.Get("dataset_query_filter", "schema", "errors"),
			},
		},
	}
}

func datasetQueryFilterToResourceData(filter *rest.DatasetQueryFilterResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	data.SetId(filter.Id)
	setResourceData("oid", filter.Oid().String())

	setResourceData("label", filter.Label)
	setResourceData("description", filter.Description)
	setResourceData("filter", filter.Filter)
	setResourceData("disabled", filter.Disabled)
	setResourceData("errors", filter.Errors)
	return diags
}

func datasetQueryFilterDefinitionFromResourceData(data *schema.ResourceData) (req *rest.DatasetQueryFilterDefinition, diags diag.Diagnostics) {
	req = &rest.DatasetQueryFilterDefinition{}

	req.Label = data.Get("label").(string)
	req.Filter = data.Get("filter").(string)
	req.Disabled = data.Get("disabled").(bool)

	if description, ok := data.GetOk("description"); ok {
		req.Description = description.(string)
	}

	return req, diags
}

func resourceDatasetQueryFilterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := datasetQueryFilterDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	datasetOid, err := oid.NewOID(data.Get("dataset").(string))
	if err != nil {
		return diag.Errorf("invalid dataset OID: %s", err.Error())
	}

	filter, err := client.CreateDatasetQueryFilter(ctx, datasetOid.Id, req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dataset query filter",
			Detail:   err.Error(),
		})
		return diags
	}
	data.SetId(filter.Id)
	return append(diags, resourceDatasetQueryFilterRead(ctx, data, meta)...)
}

func resourceDatasetQueryFilterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	datasetOid, err := oid.NewOID(data.Get("dataset").(string))
	if err != nil {
		return diag.Errorf("invalid dataset OID: %s", err.Error())
	}

	result, err := client.GetDatasetQueryFilter(ctx, datasetOid.Id, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve dataset query filter: %s", err.Error())
	}

	return datasetQueryFilterToResourceData(result, data)
}

func resourceDatasetQueryFilterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := datasetQueryFilterDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	datasetOid, err := oid.NewOID(data.Get("dataset").(string))
	if err != nil {
		return diag.Errorf("invalid dataset OID: %s", err.Error())
	}

	_, err = client.UpdateDatasetQueryFilter(ctx, datasetOid.Id, data.Id(), req)
	if err != nil {
		return diag.Errorf("failed to update dataset query filter: %s", err.Error())
	}

	return append(diags, resourceDatasetQueryFilterRead(ctx, data, meta)...)
}

func resourceDatasetQueryFilterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	datasetOid, err := oid.NewOID(data.Get("dataset").(string))
	if err != nil {
		return diag.Errorf("invalid dataset OID: %s", err.Error())
	}

	err = client.DeleteDatasetQueryFilter(ctx, datasetOid.Id, data.Id())
	if err != nil {
		return diag.Errorf("failed to delete dataset query filter: %s", err.Error())
	}
	return diags
}
