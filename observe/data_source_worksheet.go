package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/binding"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceWorksheet() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches metadata for an existing Observe worksheet.",
		ReadContext: dataSourceWorksheetRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: diffSuppressWorkspace,
				Deprecated:       "workspace is no longer required and will be ignored. It may be removed in a future version.",
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaWorksheetWorkspaceDescription,
			},
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateID(),
				Description:      "Worksheet ID.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetOIDDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetNameDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetIconDescription,
			},
			"queries": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetJSONDescription,
			},
			"object_tags": objectTagsSchemaFieldComputed(),
			"_bindings": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaWorksheetBindingsDescription,
			},
		},
	}
}

func dataSourceWorksheetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		id     = data.Get("id").(string)
	)

	ws, err := client.GetWorksheet(ctx, id)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(ws.Id)

	diags = worksheetToResourceData(ws, data)
	if diags.HasError() {
		return diags
	}

	if client.ExportMode {
		if err := escapeExportedStrings(data, dataSourceWorksheet().Schema); err != nil {
			return diag.FromErr(err)
		}
	}

	if client.ExportObjectBindings {
		err := generateWorksheetBindings(ctx, ws, data, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return diags
}

// Generates bindings for use in cross-tenant exports of worksheets. See binding.go for details.
func generateWorksheetBindings(ctx context.Context, ws *gql.Worksheet, data *schema.ResourceData, client *observe.Client) error {
	bindFor := binding.NewKindSet(binding.KindDataset, binding.KindWorkspace)
	gen, err := binding.NewGenerator(ctx, binding.KindWorksheet, ws.Label, client, bindFor)
	if err != nil {
		return fmt.Errorf("failed to initialize binding generator: %w", err)
	}

	workspaceRef, _ := gen.TryBindOid(oid.WorkspaceOid(ws.WorkspaceId))
	if err := data.Set("workspace", workspaceRef); err != nil {
		return err
	}

	queriesJson := data.Get("queries").(string)
	if queriesJson != "" {
		queriesWithReferences, err := gen.GenerateJson([]byte(queriesJson))
		if err != nil {
			return fmt.Errorf("failed to generate bindings for field 'queries': %w", err)
		}
		if err := data.Set("queries", string(queriesWithReferences)); err != nil {
			return err
		}
	}

	bindingsJson, err := gen.GetBindingsJson()
	if err != nil {
		return err
	}
	if err := data.Set("_bindings", string(bindingsJson)); err != nil {
		return err
	}
	return nil
}
