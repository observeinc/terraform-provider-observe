package observe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceInboundShareTable() *schema.Resource {
	return &schema.Resource{
		Description:   "Tracks a table from an inbound Snowflake share and creates an associated Observe dataset.",
		CreateContext: resourceInboundShareTableCreate,
		ReadContext:   resourceInboundShareTableRead,
		UpdateContext: resourceInboundShareTableUpdate,
		DeleteContext: resourceInboundShareTableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceInboundShareTableImport,
		},
		Schema: map[string]*schema.Schema{
			"share_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeInboundShare),
				Description:      "The OID of the share containing the table.",
			},
			"table_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the table in the share.",
			},
			"schema_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The schema name containing the table.",
			},
			"dataset_label": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDatasetName(),
				Description:      "The label for the created Observe dataset.",
			},
			"dataset_kind": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice([]string{"Table", "Event", "Resource", "Interval"}, false),
				Description:      "The kind of dataset to create. Accepted values: `Table`, `Event`, `Resource`, `Interval`.",
			},
			"valid_from_field": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The field to use as the valid_from timestamp (for Event/Interval datasets).",
			},
			"valid_to_field": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The field to use as the valid_to timestamp (for Interval datasets).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the tracked table.",
			},
			"field_mapping": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Field mapping configuration for type conversions (e.g., timestamp conversions).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the field to map.",
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The target data type (timestamp, duration, int64, string, etc.).",
						},
						"conversion": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The conversion method (Direct, MillisecondsToTimestamp, etc.).",
						},
					},
				},
			},
			// Computed outputs
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"table_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the tracked table.",
			},
			"dataset_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the created dataset.",
			},
			"dataset_oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The OID of the created dataset.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operational status of the tracked table.",
			},
			"full_table_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full path to the table (schema/table).",
			},
		},
	}
}

func newTrackTableRequest(data *schema.ResourceData) (*rest.TrackTableRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	req := &rest.TrackTableRequest{
		TableName:    data.Get("table_name").(string),
		SchemaName:   data.Get("schema_name").(string),
		DatasetLabel: data.Get("dataset_label").(string),
		DatasetKind:  data.Get("dataset_kind").(string),
	}

	if v, ok := data.GetOk("valid_from_field"); ok {
		req.ValidFromField = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("valid_to_field"); ok {
		req.ValidToField = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		req.Description = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("field_mapping"); ok {
		req.SchemaMapping = make(map[string]rest.FieldMapping)
		mappingSet := v.(*schema.Set)
		for _, item := range mappingSet.List() {
			m := item.(map[string]interface{})
			fieldName := m["field"].(string)
			req.SchemaMapping[fieldName] = rest.FieldMapping{
				Type:       m["type"].(string),
				Conversion: m["conversion"].(string),
			}
		}
	}

	return req, diags
}

func inboundShareTableToResourceData(result *rest.TrackTableResponse, data *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics

	table := result.Table
	dataset := result.Dataset

	if err := data.Set("oid", table.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("table_id", table.Id); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset_id", dataset.Id); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	datasetOid := oid.OID{Id: dataset.Id, Type: oid.TypeDataset}
	if err := data.Set("dataset_oid", datasetOid.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("status", table.Status); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("full_table_path", table.FullTablePath); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// Update dataset_label from the actual dataset
	if err := data.Set("dataset_label", dataset.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceInboundShareTableCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	shareOid, err := oid.NewOID(data.Get("share_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	req, diags := newTrackTableRequest(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.TrackTable(ctx, shareOid.Id, req)
	if err != nil {
		return diag.Errorf("failed to track table: %v", err)
	}

	// Set the resource ID to the table ID
	data.SetId(result.Table.Id)

	return append(diags, inboundShareTableToResourceData(result, data)...)
}

func resourceInboundShareTableRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	shareOid, err := oid.NewOID(data.Get("share_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	result, err := client.GetInboundShareTable(ctx, shareOid.Id, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// Table has been deleted outside Terraform
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve tracked table: %v", err)
	}

	return inboundShareTableToResourceData(result, data)
}

func resourceInboundShareTableUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Will be implemented in Commit 7
	return diag.Errorf("update not yet implemented - this is a placeholder for Commit 7")
}

func resourceInboundShareTableDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Will be implemented in Commit 7
	return diag.Errorf("delete not yet implemented - this is a placeholder for Commit 7")
}

func resourceInboundShareTableImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*observe.Client)

	// Import using the table OID
	tableOid, err := oid.NewOID(data.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid OID format: %w", err)
	}

	if tableOid.Type != oid.TypeInboundShareTable {
		return nil, fmt.Errorf("expected OID type %s, got %s", oid.TypeInboundShareTable, tableOid.Type)
	}

	// Find which share contains this table by listing all shares
	shares, err := client.ListShares(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list shares during import: %w", err)
	}

	for _, share := range shares.Shares {
		result, err := client.GetInboundShareTable(ctx, share.Id, tableOid.Id)
		if err == nil {
			// Found the table in this share
			if err := data.Set("share_id", share.Oid().String()); err != nil {
				return nil, err
			}
			data.SetId(result.Table.Id)

			// Set required fields from the API response
			if err := data.Set("table_name", result.Table.TableName); err != nil {
				return nil, err
			}
			if err := data.Set("schema_name", result.Table.SchemaName); err != nil {
				return nil, err
			}
			if err := data.Set("dataset_label", result.Dataset.Label); err != nil {
				return nil, err
			}
			if err := data.Set("dataset_kind", result.Dataset.Kind); err != nil {
				return nil, err
			}

			return []*schema.ResourceData{data}, nil
		}
	}

	return nil, fmt.Errorf("could not find table %s in any share", tableOid.Id)
}


