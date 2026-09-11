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

const (
	schemaDatastreamWorkspaceDescription   = "OID of workspace datastream is contained in."
	schemaDatastreamNameDescription        = "Datastream name. Must be unique within workspace."
	schemaDatastreamDescriptionDescription = "Datastream description."
	schemaDatastreamIconDescription        = "Icon image."
	schemaDatastreamTypeDescription        = "Datastream type. Valid values are `Prometheus`, `OtelLogs`, `OtelMetrics`, `K8sEntity`, and `OtelTrace`. Changing this value forces Terraform to create a new datastream."
	schemaDatastreamOIDDescription         = "The Observe ID for datastream."
	schemaDatastreamDatasetDescription     = "The Observe ID for datastream origin dataset."
)

var datastreamTypes = []string{"Prometheus", "OtelLogs", "OtelMetrics", "K8sEntity", "OtelTrace"}

func resourceDatastream() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a datastream. A datastream represents a source of data being ingested into Observe.",
		CreateContext: resourceDatastreamCreate,
		ReadContext:   resourceDatastreamRead,
		UpdateContext: resourceDatastreamUpdate,
		DeleteContext: resourceDatastreamDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				DiffSuppressFunc: diffSuppressWorkspace,
				Deprecated:       "workspace is no longer required and will be ignored. It may be removed in a future version.",
				Description:      schemaDatastreamWorkspaceDescription,
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      schemaDatastreamNameDescription,
				ValidateDiagFunc: validateDatastreamName(),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDatastreamDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDatastreamIconDescription,
			},
			"type": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice(datastreamTypes, false),
				Description:      schemaDatastreamTypeDescription,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamOIDDescription,
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamDatasetDescription,
			},
		},
	}
}

func newDatastreamConfig(data *schema.ResourceData) (*gql.DatastreamInput, diag.Diagnostics) {
	input := &gql.DatastreamInput{
		Name: data.Get("name").(string),
	}

	{
		// always reset to empty string if description not set
		input.Description = stringPtr(data.Get("description").(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("type"); ok {
		directWrite := &gql.DatastreamDirectWriteInput{}
		enabled := true
		switch v.(string) {
		case "Prometheus":
			directWrite.Prometheus = &enabled
		case "OtelLogs":
			directWrite.OtelLogs = &enabled
		case "OtelMetrics":
			directWrite.OtelMetrics = &enabled
		case "K8sEntity":
			directWrite.K8sEntity = &enabled
		case "OtelTrace":
			directWrite.OtelTrace = &enabled
		default:
			return nil, diag.Errorf("unsupported datastream type %q", v.(string))
		}
		input.DirectWrite = directWrite
	}

	return input, nil
}

func datastreamToResourceData(d *gql.Datastream, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Description != nil {
		if err := data.Set("description", d.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.DatasetId != nil {
		if err := data.Set("dataset", oid.DatasetOid(*d.DatasetId).String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	} else if datasetID := datastreamDirectWritePrimaryDatasetID(d.DirectWrite); datasetID != "" {
		if err := data.Set("dataset", oid.DatasetOid(datasetID).String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func resourceDatastreamToResourceData(d *gql.Datastream, data *schema.ResourceData) (diags diag.Diagnostics) {
	diags = datastreamToResourceData(d, data)
	if typeName := datastreamDirectWriteType(d.DirectWrite); typeName != "" {
		if err := data.Set("type", typeName); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	return diags
}

func datastreamDirectWriteType(directWrite *gql.DatastreamDirectWrite) string {
	if directWrite == nil {
		return ""
	}

	types := make([]string, 0, 5)
	if directWrite.Prometheus != nil {
		types = append(types, "Prometheus")
	}
	if directWrite.OtelLogs != nil {
		types = append(types, "OtelLogs")
	}
	if directWrite.OtelMetrics != nil {
		types = append(types, "OtelMetrics")
	}
	if directWrite.K8sEntity != nil {
		types = append(types, "K8sEntity")
	}
	if directWrite.OtelTrace != nil {
		types = append(types, "OtelTrace")
	}
	if len(types) == 1 {
		return types[0]
	}
	return ""
}

func datastreamDirectWritePrimaryDatasetID(directWrite *gql.DatastreamDirectWrite) string {
	switch datastreamDirectWriteType(directWrite) {
	case "Prometheus":
		return directWrite.Prometheus.DatasetId
	case "OtelLogs":
		return directWrite.OtelLogs.DatasetId
	case "OtelMetrics":
		return directWrite.OtelMetrics.DatasetId
	case "K8sEntity":
		return directWrite.K8sEntity.DatasetId
	case "OtelTrace":
		return directWrite.OtelTrace.SpanDatasetId
	default:
		return ""
	}
}

func resourceDatastreamCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamConfig(data)
	if diags.HasError() {
		return diags
	}

	wsid, err := client.ResolveWorkspaceID(ctx, maybeString(data.GetOk("workspace")))
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}
	result, err := client.CreateDatastream(ctx, wsid, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create datastream",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceDatastreamRead(ctx, data, meta)...)
}

func resourceDatastreamRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDatastream(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return resourceDatastreamToResourceData(result, data)
}

func resourceDatastreamUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamConfig(data)
	if diags.HasError() {
		return diags
	}
	// Nil leaves datastream types unchanged because type is ForceNew.
	config.DirectWrite = nil

	result, err := client.UpdateDatastream(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return resourceDatastreamToResourceData(result, data)
}

func resourceDatastreamDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDatastream(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete datastream: %s", err)
	}
	return diags
}
