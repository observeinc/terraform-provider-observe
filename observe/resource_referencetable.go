package observe

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const (
	schemaReferenceTableWorkspaceDescription   = "OID of the workspace the reference table is contained in."
	schemaReferenceTableNameDescription        = "The name of the reference table name. Must be unique within workspace."
	schemaReferenceTableDescriptionDescription = "Description for the reference table."
	schemaReferenceTableIconDescription        = "Icon image."
	schemaReferenceTableOIDDescription         = "The Observe ID for the reference table."
	schemaReferenceTableDatasetDescription     = "The Observe ID for the dataset managed by the reference table."
)

func resourceReferenceTable() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a reference table. A reference table represents a source of non-temporal data being ingested into Observe.",
		CreateContext: resourceReferenceTableCreate,
		ReadContext:   resourceReferenceTableRead,
		UpdateContext: resourceReferenceTableUpdate,
		DeleteContext: resourceReferenceTableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("commmon", "schema", "oid"),
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      schemaReferenceTableNameDescription,
				ValidateDiagFunc: validateReferenceTableName(),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaReferenceTableDescriptionDescription,
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaReferenceTableDatasetDescription,
			},
			"file_path": {
				Type:        schema.TypeString,
				Description: "TODO",
				Required:    true,
				// TODO: validate file path
			},
			"schema": {
				Type:     schema.TypeList,
				Optional: true,
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
					},
				},
				Description: "TODO",
			},
			"primary_key": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "TODO",
			},
		},
	}
}

func newReferenceTableConfig(data *schema.ResourceData) (*rest.ReferenceTableInput, diag.Diagnostics) {
	input := &rest.ReferenceTableInput{}

	if v, ok := data.GetOk("name"); ok {
		input.Name = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("file_path"); ok {
		input.UploadFilePath = v.(string)
	}

	if v, ok := data.GetOk("schema"); ok {
		var fields []gql.DatasetFieldDefInput
		schema := v.([](map[string]string))
		for _, col := range schema {
			fields = append(fields, gql.DatasetFieldDefInput{
				Name: col["name"],
				Type: gql.DatasetFieldTypeInput{
					Rep: col["type"],
				},
				// TODO: isEnum
				IsSearchable: boolPtr(true),
			})
		}
		input.Schema = fields
	}

	if v, ok := data.GetOk("primary_key"); ok {
		input.PrimaryKey = v.([]string)
	}

	return input, nil
}

func referenceTableToResourceData(d *gql.ReferenceTable, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Description != nil {
		if err := data.Set("description", d.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("dataset", oid.DatasetOid(d.DatasetID).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceReferenceTableCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newReferenceTableConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateReferenceTable(ctx, id.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create reference table",
			Detail:   err.Error(),
		})
		return diags
	}

	// TODO: may need to set additional fields?
	data.SetId(result.Id)
	return append(diags, resourceReferenceTableRead(ctx, data, meta)...)
}

func resourceReferenceTableRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetReferenceTable(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve reference table [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}
	return referenceTableToResourceData(result, data)
}

func resourceReferenceTableUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newReferenceTableConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateReferenceTable(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update reference table [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return referenceTableToResourceData(result, data)
}

func resourceReferenceTableDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteReferenceTable(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete reference table: %s", err)
	}
	return diags
}
