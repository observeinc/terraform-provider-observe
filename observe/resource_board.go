package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaBoardOIDDescription     = "Observe ID of Board."
	schemaBoardDatasetDescription = "OID of Dataset for which board is defined."
	schemaBoardTypeDescription    = "Type of board."
	schemaBoardNameDescription    = "Board name."
	schemaBoardJSONDescription    = "JSON representation of board contents."
)

func resourceBoard() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBoardCreate,
		ReadContext:   resourceBoardRead,
		UpdateContext: resourceBoardUpdate,
		DeleteContext: resourceBoardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaBoardOIDDescription,
			},
			"dataset": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDataset),
				DiffSuppressFunc: diffSuppressVersion,
				Description:      schemaBoardDatasetDescription,
			},
			"type": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateEnums(observe.BoardTypes),
				Description:      describeEnums(observe.BoardTypes, schemaBoardTypeDescription),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaBoardNameDescription,
			},
			"json": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateStringIsJSON,
				Description:      schemaBoardJSONDescription,
			},
		},
	}
}

func newBoardConfig(data *schema.ResourceData) (config *observe.BoardConfig, diags diag.Diagnostics) {
	config = &observe.BoardConfig{
		Name: data.Get("name").(string),
		JSON: data.Get("json").(string),
	}

	return config, diags
}

func boardToResourceData(b *observe.Board, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", b.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("type", toSnake(b.Type.String())); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset", b.Dataset.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("json", b.Config.JSON); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", b.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceBoardCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		dataset, _ = observe.NewOID(data.Get("dataset").(string))
	)
	config, diags := newBoardConfig(data)
	if diags.HasError() {
		return diags
	}

	boardTypeStr := data.Get("type").(string)
	boardType := observe.BoardType(toCamel(boardTypeStr))

	result, err := client.CreateBoard(ctx, dataset, boardType, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create board",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
	return append(diags, resourceBoardRead(ctx, data, meta)...)
}

func resourceBoardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetBoard(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dataset [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return boardToResourceData(result, data)
}

func resourceBoardUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBoardConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateBoard(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update board [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return boardToResourceData(result, data)
}

func resourceBoardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteBoard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete board: %s", err)
	}
	return diags
}
