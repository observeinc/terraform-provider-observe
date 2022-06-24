package observe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
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
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
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
				DiffSuppressFunc: diffSuppressStageQueryInput,
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

func newWorksheetConfig(data *schema.ResourceData) (input *gql.WorksheetInput, diags diag.Diagnostics) {
	input = &gql.WorksheetInput{
		Label: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.Icon = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("queries"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.Stages); err != nil {
			diagErr := fmt.Errorf("failed to parse 'queries' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}
	return input, diags
}

func worksheetToResourceData(d *gql.Worksheet, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Icon != nil {
		if err := data.Set("icon_url", d.Icon); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Stages != nil {
		// Hack hack hack hack hack
		for i, stage := range d.Stages {
			if stage.Id != nil && *stage.Id == "" {
				d.Stages[i].Id = nil
			}
			for j, input := range stage.Input {
				if input.StageId != nil && *input.StageId == "" {
					d.Stages[i].Input[j].StageId = nil
				}
			}
			if stage.Params != nil && *stage.Params == types.JsonObject("null") {
				d.Stages[i].Params = nil
			} else if stage.Params != nil && string(*stage.Params) == "" {
				d.Stages[i].Params = nil
			}
		}
		if stagesRaw, err := json.Marshal(d.Stages); err != nil {
			diagErr := fmt.Errorf("failed to parse 'stages' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if err := data.Set("queries", string(stagesRaw)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
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

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateWorksheet(ctx, id.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create worksheet",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
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

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateWorksheet(ctx, data.Id(), id.Id, config)
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
