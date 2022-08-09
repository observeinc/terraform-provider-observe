package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var sourceDatasetFieldResource = &schema.Resource{
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
			// TODO(luke): Do not use DefaultFunc instead of Default.
			// sourceDatasetToResourceData assumes that we are using Default.
			Default: false,
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
}

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
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
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
			"source_update_table_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"valid_from_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"batch_seq_field": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_insert_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"field": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     sourceDatasetFieldResource,
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
			"freshness": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressDuration,
			},
		},
	}
}

func newSourceDatasetConfig(data *schema.ResourceData) (*gql.DatasetDefinitionInput, *gql.SourceTableDefinitionInput, error) {
	description := data.Get("description").(string)
	input := gql.DatasetDefinitionInput{
		Dataset: gql.DatasetInput{
			Label: data.Get("name").(string),
			// always reset to empty string if description not set
			Description: &description,
		},
	}
	if v, ok := data.GetOk("icon_url"); ok {
		input.Dataset.IconUrl = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("freshness"); ok {
		// we already validated in schema
		freshnessParsed, _ := time.ParseDuration(v.(string))
		input.Dataset.FreshnessDesired = types.Int64Scalar(freshnessParsed).Ptr()
	}

	sourceUpdateTableName := data.Get("source_update_table_name").(string)
	validFromField := data.Get("valid_from_field").(string)
	sourceInput := gql.SourceTableDefinitionInput{
		Schema:                data.Get("schema").(string),
		TableName:             data.Get("table_name").(string),
		SourceUpdateTableName: &sourceUpdateTableName,
		ValidFromField:        &validFromField,
	}
	if v, ok := data.GetOk("batch_seq_field"); ok {
		sourceInput.BatchSeqField = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("is_insert_only"); ok {
		sourceInput.IsInsertOnly = boolPtr(v.(bool))
	}

	for _, fieldRaw := range data.Get("field").(*schema.Set).List() {
		field := fieldRaw.(map[string]interface{})
		isEnum := field["is_enum"].(bool)
		isSearchable := field["is_searchable"].(bool)
		isHidden := field["is_hidden"].(bool)
		isConst := field["is_const"].(bool)
		isMetric := field["is_metric"].(bool)
		input.Schema = append(input.Schema, gql.DatasetFieldDefInput{
			Name: field["name"].(string),
			Type: gql.DatasetFieldTypeInput{
				Rep: field["type"].(string),
			},
			IsEnum:       &isEnum,
			IsSearchable: &isSearchable,
			IsHidden:     &isHidden,
			IsConst:      &isConst,
			IsMetric:     &isMetric,
		})
		sourceInput.Fields = append(sourceInput.Fields, gql.SourceTableFieldDefinitionInput{
			Name:    field["name"].(string),
			SqlType: field["sql_type"].(string),
		})
	}

	return &input, &sourceInput, nil
}

func sourceDatasetToResourceData(d *gql.Dataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("schema", d.SourceTable.Schema); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("table_name", d.SourceTable.TableName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.SourceTable.SourceUpdateTableName != nil {
		if err := data.Set("source_update_table_name", d.SourceTable.SourceUpdateTableName); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.SourceTable.BatchSeqField != nil {
		if err := data.Set("batch_seq_field", d.SourceTable.BatchSeqField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("is_insert_only", d.SourceTable.IsInsertOnly); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.SourceTable.ValidFromField != nil {
		if err := data.Set("valid_from_field", d.SourceTable.ValidFromField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	fieldDefsMap := make(map[string]gql.DatasetTypedefDefObjectTypedefFieldsObjectFieldDef)
	for _, d := range d.Typedef.Def.Fields {
		fieldDefsMap[d.Name] = d
	}

	fields := schema.NewSet(schema.HashResource(sourceDatasetFieldResource), nil)
	for _, field := range d.SourceTable.Fields {
		fieldDef := fieldDefsMap[field.Name]
		f := map[string]interface{}{
			"name":          field.Name,
			"type":          fieldDef.Type.Rep,
			"sql_type":      field.SqlType,
			"is_enum":       sourceDatasetFieldResource.Schema["is_enum"].Default,
			"is_searchable": sourceDatasetFieldResource.Schema["is_searchable"].Default,
			"is_hidden":     sourceDatasetFieldResource.Schema["is_hidden"].Default,
			"is_const":      sourceDatasetFieldResource.Schema["is_const"].Default,
			"is_metric":     sourceDatasetFieldResource.Schema["is_metric"].Default,
		}

		if fieldDef.IsEnum != nil {
			f["is_enum"] = *fieldDef.IsEnum
		}
		if fieldDef.IsSearchable != nil {
			f["is_searchable"] = *fieldDef.IsSearchable
		}
		if fieldDef.IsHidden != nil {
			f["is_hidden"] = *fieldDef.IsHidden
		}
		if fieldDef.IsConst != nil {
			f["is_const"] = *fieldDef.IsConst
		}
		if fieldDef.IsMetric != nil {
			f["is_metric"] = *fieldDef.IsMetric
		}

		fields.Add(f)
	}
	if err := data.Set("field", fields); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.FreshnessDesired != nil {
		if err := data.Set("freshness", d.FreshnessDesired.Duration().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
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

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceSourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	input, sourceInput, err := newSourceDatasetConfig(data)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateSourceDataset(ctx, workspace.Id, input, sourceInput)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dataset",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
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
	input, sourceInput, err := newSourceDatasetConfig(data)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateSourceDataset(ctx, workspace.Id, data.Id(), input, sourceInput)
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
