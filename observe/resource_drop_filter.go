package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
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
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
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
			},
		},
	}
}

func newIngestFilterConfig(data *schema.ResourceData) (input *gql.IngestFilterInput, diags diag.Diagnostics) {
	var (
		name               = data.Get("name").(string)
		pipeline           = data.Get("pipeline").(string)
		dropRate           = data.Get("drop_rate").(float64)
		enabled            = data.Get("enabled").(bool)
		sourceDatasetID, _ = oid.NewOID(data.Get("source_dataset").(string))
	)
	input = &gql.IngestFilterInput{
		Name:            name,
		Pipeline:        pipeline,
		DropRate:        dropRate,
		Enabled:         enabled,
		SourceDatasetID: sourceDatasetID.Id,
	}
	return input, diags
}

func ingestFilterToResourceData(filter *gql.IngestFilter, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(filter.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", oid.IngestFilterOid(filter.Id).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("source_dataset", oid.DatasetOid(filter.SourceDatasetID).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", filter.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("drop_rate", filter.DropRate); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("pipeline", filter.Pipeline); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceIngestFilterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newIngestFilterConfig(data)
	workspace, _ := oid.NewOID(data.Get("workspace").(string))

	if diags.HasError() {
		return diags
	}
	filter, err := client.CreateIngestFilter(ctx, workspace.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create drop filter",
			Detail:   err.Error(),
		})
		return diags
	}
	if len(filter.Errors) > 0 {
		for _, v := range filter.Errors {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "failed to create drop filter",
				Detail:   v.GetMessage(),
			})
		}
		return diags
	}

	data.SetId(filter.Id)

	return append(diags, resourceIngestFilterRead(ctx, data, meta)...)
}

func resourceIngestFilterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	filter, err := client.GetIngestFilter(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve drop filter [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}
	if len(filter.Errors) > 0 {
		for _, v := range filter.Errors {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("failed to retrieve drop filter [id=%s]", data.Id()),
				Detail:   v.GetMessage(),
			})
		}
		return diags
	}
	return ingestFilterToResourceData(filter, data)
}

func resourceIngestFilterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newIngestFilterConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateIngestFilter(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update drop filter [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}
	if len(result.Errors) > 0 {
		for _, v := range result.Errors {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("failed to update drop filter [id=%s]", data.Id()),
				Detail:   v.GetMessage(),
			})
		}
		return diags
	}

	return ingestFilterToResourceData(result, data)
}

func resourceIngestFilterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteIngestFilter(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete drop filter: %s", err)
	}
	return diags
}
