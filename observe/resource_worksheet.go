package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaWorksheetWorkspaceDescription = "OID of workspace worksheet is contained in."
	schemaWorksheetNameDescription      = "Worksheet name. Must be unique within workspace."
	schemaWorksheetIconDescription      = "Icon image."
	schemaWorksheetJSONDescription      = "Worksheet definition in JSON format."
	schemaWorksheetOIDDescription       = "The Observe ID for worksheet."
)

func resourceWorksheet() *schema.Resource {
	return &schema.Resource{
		Description:   "",
		CreateContext: resourceWorksheetCreate,
		ReadContext:   resourceWorksheetRead,
		UpdateContext: resourceWorksheetUpdate,
		DeleteContext: resourceWorksheetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
				Description:      schemaWorksheetWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaWorksheetNameDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaWorksheetIconDescription,
			},
			"queries": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaWorksheetJSONDescription,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetOIDDescription,
			},
		},
	}
}

func newWorksheetConfig(data *schema.ResourceData) (config *observe.WorksheetConfig, diags diag.Diagnostics) {
	config = &observe.WorksheetConfig{
		Name: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.IconURL = &icon
	}

	if v, ok := data.GetOk("queries"); ok {
		data := v.(string)
		config.Queries = &data
	}
	return config, diags
}

func worksheetToResourceData(d *observe.Worksheet, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.IconURL != nil {
		if err := data.Set("icon_url", d.Config.IconURL); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.Queries != nil {
		if err := data.Set("queries", d.Config.Queries); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceWorksheetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newWorksheetConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateWorksheet(ctx, oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create worksheet",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
	return append(diags, resourceWorksheetRead(ctx, data, meta)...)
}

func resourceWorksheetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetWorksheet(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve worksheet [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return worksheetToResourceData(result, data)
}

func resourceWorksheetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newWorksheetConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateWorksheet(ctx, data.Id(), oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update worksheet [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return worksheetToResourceData(result, data)
}

func resourceWorksheetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteWorksheet(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete worksheet: %s", err)
	}
	return diags
}
