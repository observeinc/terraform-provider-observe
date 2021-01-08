package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceSourceDataset() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSourceDatasetCreate,
		ReadContext:   resourceSourceDatasetRead,
		UpdateContext: resourceSourceDatasetUpdate,
		DeleteContext: resourceSourceDatasetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if datasetRecomputeOID(d) {
				return d.SetNewComputed("oid")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"schema": {
				Type:     schema.TypeString,
				Required: true,
			},
			"table_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			// Note: source_update_table_name, batch_seq_field, and
			// valid_from_field are required in terraform but optional in
			// the SourceDatasetConfig struct.
			"source_update_table_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"batch_seq_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"valid_from_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"field": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"sql_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"is_enum": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"is_searchable": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"is_hidden": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"is_const": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"is_metric": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"freshness": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(i interface{}, k string) ([]string, []error) {
					s := i.(string)
					if _, err := time.ParseDuration(s); err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					o, _ := time.ParseDuration(old)
					n, _ := time.ParseDuration(new)
					return o == n
				},
			},
		},
	}
}

func newSourceDatasetConfig(data *schema.ResourceData) *observe.SourceDatasetConfig {
	config := &observe.SourceDatasetConfig{
		Name:      data.Get("name").(string),
		Schema:    data.Get("schema").(string),
		TableName: data.Get("table_name").(string),
	}

	sourceUpdateTableFqn := data.Get("source_update_table_name").(string)
	config.SourceUpdateTableName = &sourceUpdateTableFqn
	batchSeqField := data.Get("batch_seq_field").(string)
	config.BatchSeqField = &batchSeqField
	validFromField := data.Get("valid_from_field").(string)
	config.ValidFromField = &validFromField

	for i := range data.Get("field").([]interface{}) {
		var field observe.SourceDatasetFieldConfig
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.name", i)); ok {
			field.Name = v.(string)
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.type", i)); ok {
			field.Type = v.(string)
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.sql_type", i)); ok {
			field.SqlType = v.(string)
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.is_enum", i)); ok {
			b := v.(bool)
			field.IsEnum = &b
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.is_searchable", i)); ok {
			b := v.(bool)
			field.IsSearchable = &b
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.is_hidden", i)); ok {
			b := v.(bool)
			field.IsHidden = &b
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.is_const", i)); ok {
			b := v.(bool)
			field.IsConst = &b
		}
		if v, ok := data.GetOk(fmt.Sprintf("field.%d.is_metric", i)); ok {
			b := v.(bool)
			field.IsMetric = &b
		}

		config.Fields = append(config.Fields, field)
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.IconURL = &icon
	}
	{
		// always reset to empty string if description not set
		description := data.Get("description").(string)
		config.Description = &description
	}
	if v, ok := data.GetOk("freshness"); ok {
		t, _ := time.ParseDuration(v.(string))
		config.Freshness = &t
	}

	return config
}

func sourceDatasetToResourceData(d *observe.SourceDataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("schema", d.Config.Schema); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("table_name", d.Config.TableName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.SourceUpdateTableName != nil {
		if err := data.Set("source_update_table_name", d.Config.SourceUpdateTableName); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.BatchSeqField != nil {
		if err := data.Set("batch_seq_field", d.Config.BatchSeqField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.ValidFromField != nil {
		if err := data.Set("valid_from_field", d.Config.ValidFromField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	fields := make([]interface{}, len(d.Config.Fields))
	for i, field := range d.Config.Fields {
		f := map[string]interface{}{
			"name":     field.Name,
			"type":     field.Type,
			"sql_type": field.SqlType,
		}
		if field.IsEnum != nil {
			f["is_enum"] = field.IsEnum
		}
		if field.IsHidden != nil {
			f["is_hidden"] = field.IsHidden
		}
		if field.IsConst != nil {
			f["is_const"] = field.IsConst
		}
		if field.IsMetric != nil {
			f["is_metric"] = field.IsMetric
		}

		fields[i] = f
	}

	if d.Config.Freshness != nil {
		if err := data.Set("freshness", d.Config.Freshness.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
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

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceSourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config := newSourceDatasetConfig(data)

	workspace, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateSourceDataset(ctx, workspace.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dataset",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
	return append(diags, resourceSourceDatasetRead(ctx, data, meta)...)
}

func resourceSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetSourceDataset(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dataset [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return sourceDatasetToResourceData(result, data)
}

func resourceSourceDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config := newSourceDatasetConfig(data)

	workspace, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateSourceDataset(ctx, workspace.ID, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update dataset [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return sourceDatasetToResourceData(result, data)
}

func resourceSourceDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDataset(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dataset: %s", err)
	}
	return diags
}
