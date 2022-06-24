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
	schemaDatastreamOIDDescription         = "The Observe ID for datastream."
	schemaDatastreamDatasetDescription     = "The Observe ID for datastream origin dataset."
)

func resourceDatastream() *schema.Resource {
	return &schema.Resource{
		Description:   "",
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
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaDatastreamWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDatastreamNameDescription,
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

	return input, nil
}

func datastreamToResourceData(d *gql.Datastream, data *schema.ResourceData) (diags diag.Diagnostics) {
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

	if err := data.Set("dataset", oid.DatasetOid(d.DatasetId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourceDatastreamCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateDatastream(ctx, id.Id, config)
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
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return datastreamToResourceData(result, data)
}

func resourceDatastreamUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateDatastream(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return datastreamToResourceData(result, data)
}

func resourceDatastreamDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDatastream(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete datastream: %s", err)
	}
	return diags
}
