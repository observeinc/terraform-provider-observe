package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
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
		Description:        "Manages an Observe board.",
		DeprecationMessage: "Boards have been deprecated in favor of dashboards, which can define their own stages for futher processing of datasets.",
		CreateContext:      resourceBoardCreate,
		ReadContext:        resourceBoardRead,
		UpdateContext:      resourceBoardUpdate,
		DeleteContext:      resourceBoardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaBoardOIDDescription,
			},
			"dataset": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressVersion,
				Description:      schemaBoardDatasetDescription,
			},
			"type": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllBoardType),
				Description:      describeEnums(gql.AllBoardType, schemaBoardTypeDescription),
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
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaBoardJSONDescription,
			},
		},
	}
}

func newBoardInput(data *schema.ResourceData) (input *gql.BoardInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	board := types.JsonObject(data.Get("json").(string))
	input = &gql.BoardInput{
		Name:  &name,
		Board: &board,
	}

	return input, diags
}

func boardToResourceData(b *gql.Board, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", b.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("type", toSnake(string(b.Type))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset", oid.DatasetOid(b.DatasetId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("json", b.BoardJson); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", b.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceBoardCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		dataset, _ = oid.NewOID(data.Get("dataset").(string))
	)
	config, diags := newBoardInput(data)
	if diags.HasError() {
		return diags
	}

	boardTypeStr := data.Get("type").(string)
	boardType := gql.BoardType(toCamel(boardTypeStr))

	result, err := client.CreateBoard(ctx, dataset.Id, boardType, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create board",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceBoardRead(ctx, data, meta)...)
}

func resourceBoardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetBoard(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve board [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return boardToResourceData(result, data)
}

func resourceBoardUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBoardInput(data)
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
