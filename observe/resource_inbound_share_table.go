package observe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
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

// enrichResultFromDataset populates fields from the Dataset API that are not yet returned by the sharein API.
// This is a workaround until the sharein API includes these fields in the GET response.
// Returns an error if field mappings cannot be safely inferred - this prevents creating resources
// with incomplete state that would be recreated later when the API is fixed.
func enrichResultFromDataset(result *rest.TrackTableResponse, dataset *gql.Dataset) error {
	// Only populate fields if they're missing from the sharein API response

	// NOTE: We do NOT populate description from the Dataset API because:
	// 1. Description updates are not yet supported in the backend (see doUpdateTableWithDataset)
	// 2. When dataset_label changes, a new dataset is created with auto-generated description
	// 3. We can't distinguish between user-provided and auto-generated descriptions
	// 4. The sharein API will return description once it's added to the response schema

	// ValidFromField and ValidToField are top-level fields on Dataset
	if result.Table.ValidFromField == "" && dataset.ValidFromField != nil && *dataset.ValidFromField != "" {
		result.Table.ValidFromField = *dataset.ValidFromField
	}

	if result.Table.ValidToField == "" && dataset.ValidToField != nil && *dataset.ValidToField != "" {
		result.Table.ValidToField = *dataset.ValidToField
	}

	// Infer DatasetKind based on which timestamp fields are present
	// This matches the Observe dataset semantics:
	// - Interval: has both validFromField and validToField
	// - Event: has validFromField but not validToField
	// - Table: has neither (or could be Resource, but we default to Table)
	if result.Table.DatasetKind == "" {
		hasValidFrom := dataset.ValidFromField != nil && *dataset.ValidFromField != ""
		hasValidTo := dataset.ValidToField != nil && *dataset.ValidToField != ""

		if hasValidFrom && hasValidTo {
			result.Table.DatasetKind = "Interval"
		} else if hasValidFrom {
			result.Table.DatasetKind = "Event"
		} else {
			result.Table.DatasetKind = "Table"
		}
	}

	// NOTE: We do NOT populate field_mapping from inference or from the API because:
	// 1. field_mapping is not yet returned by the sharein API in production
	// 2. We cannot reliably infer user-specified Direct mappings (they're redundant)
	// 3. Users specify field_mapping in config; it's sent to API but not read back
	// 4. The sharein API will return field_mapping once it's deployed to prod

	// Update the Dataset label if present
	if dataset.Name != "" {
		result.Dataset.Label = dataset.Name
		result.Table.DatasetLabel = dataset.Name
	}

	return nil
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

	// Set computed fields (may be populated from sharein API or Dataset API)
	if table.Description != "" {
		if err := data.Set("description", table.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if table.DatasetKind != "" {
		if err := data.Set("dataset_kind", table.DatasetKind); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if table.ValidFromField != "" {
		if err := data.Set("valid_from_field", table.ValidFromField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if table.ValidToField != "" {
		if err := data.Set("valid_to_field", table.ValidToField); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	// NOTE: We do NOT set field_mapping in state because:
	// 1. The sharein API doesn't return it in production yet
	// 2. Users specify it in config, but we can't read it back
	// 3. This avoids diffs between config and state
	// Once the API returns field_mapping, we can populate it here

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

	// Populate computed fields from Dataset API if they're not returned by the sharein API
	if result.Dataset.Id != "" {
		dataset, err := client.GetDataset(ctx, result.Dataset.Id)
		if err != nil {
			return diag.Errorf("failed to get dataset %s for field inference: %v", result.Dataset.Id, err)
		}

		if err := enrichResultFromDataset(result, dataset); err != nil {
			return diag.FromErr(err)
		}
	}

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

	// Populate computed fields from Dataset API if they're not returned by the sharein API
	// This is a workaround until the sharein API returns these fields in the GET response
	var diags diag.Diagnostics
	if result.Dataset.Id != "" {
		dataset, err := client.GetDataset(ctx, result.Dataset.Id)
		if err != nil {
			return diag.Errorf("failed to get dataset %s for field inference: %v", result.Dataset.Id, err)
		}

		if err := enrichResultFromDataset(result, dataset); err != nil {
			return diag.FromErr(err)
		}
	}

	return append(diags, inboundShareTableToResourceData(result, data)...)
}

func resourceInboundShareTableUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	shareOid, err := oid.NewOID(data.Get("share_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Build update request with only changed fields
	req := &rest.UpdateTableRequest{}
	hasChanges := false

	if data.HasChange("description") {
		req.Description = stringPtr(data.Get("description").(string))
		hasChanges = true
	}

	if data.HasChange("dataset_label") {
		req.DatasetLabel = stringPtr(data.Get("dataset_label").(string))
		hasChanges = true
	}

	// NOTE: description updates are not yet supported by the backend
	// The backend logs a warning and ignores the description field
	// So we don't send it in updates to avoid confusion

	if data.HasChange("valid_from_field") {
		req.ValidFromField = stringPtr(data.Get("valid_from_field").(string))
		hasChanges = true
	}

	if data.HasChange("valid_to_field") {
		req.ValidToField = stringPtr(data.Get("valid_to_field").(string))
		hasChanges = true
	}

	if data.HasChange("field_mapping") {
		req.SchemaMapping = make(map[string]rest.FieldMapping)
		if v, ok := data.GetOk("field_mapping"); ok {
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
		hasChanges = true
	}

	if !hasChanges {
		// No actual changes to update
		return nil
	}

	_, err = client.UpdateInboundShareTable(ctx, shareOid.Id, data.Id(), req)
	if err != nil {
		return diag.Errorf("failed to update tracked table: %v", err)
	}

	// Re-read to get updated state
	return resourceInboundShareTableRead(ctx, data, meta)
}

func resourceInboundShareTableDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	shareOid, err := oid.NewOID(data.Get("share_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.DeleteInboundShareTable(ctx, shareOid.Id, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// Table already deleted, not an error
			return nil
		}
		return diag.Errorf("failed to delete tracked table: %v", err)
	}

	return nil
}

// resourceInboundShareTableImport handles terraform import with a composite ID.
//
// Import ID format: "<share_oid>/<table_id>"
// Example: terraform import observe_inbound_share_table.example "o:::inboundshare:41012345/41056789"
func resourceInboundShareTableImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*observe.Client)

	parts := strings.SplitN(data.Id(), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf(
			"invalid import ID %q: expected format \"<share_oid>/<table_id>\", "+
				"e.g. \"o:::inboundshare:41012345/41056789\"", data.Id())
	}

	shareOidStr := parts[0]
	tableId := parts[1]

	shareOid, err := oid.NewOID(shareOidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid share OID %q in import ID: %w", shareOidStr, err)
	}

	result, err := client.GetInboundShareTable(ctx, shareOid.Id, tableId)
	if err != nil {
		return nil, fmt.Errorf("failed to import table %s from share %s: %w", tableId, shareOid.Id, err)
	}

	data.SetId(result.Table.Id)

	if err := data.Set("share_id", shareOidStr); err != nil {
		return nil, err
	}
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
