package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
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

func newDatastreamConfig(data *schema.ResourceData) (*observe.DatastreamConfig, diag.Diagnostics) {
	config := &observe.DatastreamConfig{
		Name: data.Get("name").(string),
	}

	{
		// always reset to empty string if description not set
		description := data.Get("description").(string)
		config.Description = &description
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.IconURL = &icon
	}

	return config, nil
}

func datastreamToResourceData(d *observe.Datastream, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.Description != nil {
		if err := data.Set("description", d.Config.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.IconURL != nil {
		if err := data.Set("icon_url", d.Config.IconURL); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset", d.DatasetOID().String()); err != nil {
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

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateDatastream(ctx, oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create datastream",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
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
