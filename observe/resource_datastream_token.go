package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaDatastreamTokenDatastreamDescription  = "OID of datastream token is contained in."
	schemaDatastreamTokenNameDescription        = "Datastream name. Must be unique within workspace."
	schemaDatastreamTokenDescriptionDescription = "Datastream description."
	schemaDatastreamTokenDisabledDescription    = "Disable token."
	schemaDatastreamTokenOIDDescription         = "The Observe ID for datastream token."
)

func resourceDatastreamToken() *schema.Resource {
	return &schema.Resource{
		Description:   "",
		CreateContext: resourceDatastreamTokenCreate,
		ReadContext:   resourceDatastreamTokenRead,
		UpdateContext: resourceDatastreamTokenUpdate,
		DeleteContext: resourceDatastreamTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"datastream": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(observe.TypeDatastream),
				Description:      schemaDatastreamTokenDatastreamDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDatastreamTokenNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDatastreamTokenDescriptionDescription,
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: schemaDatastreamTokenDisabledDescription,
			},
			"secret": {
				Type:      schema.TypeString,
				Sensitive: true,
				Computed:  true,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatastreamTokenOIDDescription,
			},
		},
	}
}

func newDatastreamTokenConfig(data *schema.ResourceData) (*observe.DatastreamTokenConfig, diag.Diagnostics) {
	config := &observe.DatastreamTokenConfig{
		Name: data.Get("name").(string),
	}

	{
		description := data.Get("description").(string)
		config.Description = &description
	}

	{
		disabled := data.Get("disabled").(bool)
		config.Disabled = &disabled
	}

	return config, nil
}

func datastreamTokenToResourceData(d *observe.DatastreamToken, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.Description != nil {
		if err := data.Set("description", d.Config.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("disabled", d.Config.Disabled); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDatastreamTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamTokenConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("datastream").(string))
	result, err := client.CreateDatastreamToken(ctx, oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create datastream token",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)

	if err := data.Set("secret", result.Config.Secret); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return append(diags, resourceDatastreamTokenRead(ctx, data, meta)...)
}

func resourceDatastreamTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDatastreamToken(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return datastreamTokenToResourceData(result, data)
}

func resourceDatastreamTokenUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDatastreamTokenConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateDatastreamToken(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update datastream [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return datastreamTokenToResourceData(result, data)
}

func resourceDatastreamTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDatastreamToken(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete datastream: %s", err)
	}
	return diags
}
