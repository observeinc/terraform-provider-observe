package observe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceReferenceTable() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("reference_table", "description"),
		CreateContext: resourceReferenceTableCreate,
		ReadContext:   resourceReferenceTableRead,
		UpdateContext: resourceReferenceTableUpdate,
		DeleteContext: resourceReferenceTableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"label": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      descriptions.Get("reference_table", "schema", "label"),
				ValidateDiagFunc: validateReferenceTableName(),
			},
			"source_file": {
				Type:             schema.TypeString,
				Description:      descriptions.Get("reference_table", "schema", "source_file"),
				Required:         true,
				ValidateDiagFunc: validateFilePath(stringPtr(".csv")),
			},
			// checksum is used to avoid storing the entire file in the state and needing to fetch the
			// entire file every time to compare against for detecting changes.
			// See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_object
			"checksum": { // MD5 hash of source file contents
				Type:     schema.TypeString,
				Required: true,
				Description: descriptions.Get("reference_table", "schema", "checksum") +
					"Can be computed using `filemd5(\"<source_file>\")`.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("reference_table", "schema", "description"),
			},
			"schema": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
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
				Description: descriptions.Get("reference_table", "schema", "schema"),
			},
			"primary_key": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: descriptions.Get("reference_table", "schema", "primary_key"),
			},
			"label_field": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("reference_table", "schema", "label_field"),
			},
			"dataset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("reference_table", "schema", "dataset"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

func newReferenceTableConfig(data *schema.ResourceData) (input *rest.ReferenceTableInput, diags diag.Diagnostics) {
	metadataInput, diags := newReferenceTableMetadataConfig(data, false)
	if diags.HasError() {
		return nil, diags
	}

	schemaInput, diags := newReferenceTableSchemaConfig(data)
	if diags.HasError() {
		return nil, diags
	}

	input = &rest.ReferenceTableInput{
		SourceFilePath: data.Get("source_file").(string),
		Schema:         schemaInput,
		Metadata:       *metadataInput,
	}

	return input, diags
}

func newReferenceTableMetadataConfig(data *schema.ResourceData, patch bool) (input *rest.ReferenceTableMetadataInput, diags diag.Diagnostics) {
	// If we're using PATCH, then we only want to set fields that have been modified.
	// Unmodified fields should be left as nil (which are then omitted from the JSON payload).
	// If we're using POST/PUT, then nil is unused, and we use the zero value for unset fields.
	input = &rest.ReferenceTableMetadataInput{}
	if !patch || data.HasChange("label") {
		input.Label = stringPtr(data.Get("label").(string))
	}
	if !patch || data.HasChange("description") {
		input.Description = stringPtr(data.Get("description").(string))
	}
	if !patch || data.HasChange("primary_key") {
		input.PrimaryKey = &[]string{}
		for _, v := range data.Get("primary_key").([]interface{}) {
			*input.PrimaryKey = append(*input.PrimaryKey, v.(string))
		}
	}
	if !patch || data.HasChange("label_field") {
		input.LabelField = stringPtr(data.Get("label_field").(string))
	}
	return input, diags
}

func newReferenceTableSchemaConfig(data *schema.ResourceData) (input []rest.ReferenceTableSchemaInput, diags diag.Diagnostics) {
	// We're using GetRawConfig() here instead of Get() or GetOk() because we only want to include
	// the schema in the request if the user explicitly sets it in the configuration file. Get() and
	// GetOk() will return whatever's in the state file if there's no value in the config file.
	// This causes issues because "schema" is also a computed field, so the following fails:
	//    - User defines a reference table with CSV file containing 1 column (no explicit schema)
	//        - Schema is computed and set in the state file during a read.
	//    - User changes the CSV file to contain 2 columns (still no explicit schema)
	//    - The update fails since we call the API with the new 2-column file and the old 1-column schema.
	//        - We instead want to provide no schema allowing the API to re-compute it, but
	//			Get() and GetOk() will return the 1-column schema from the state file.
	rawConfig := data.GetRawConfig().AsValueMap()
	if schema, ok := rawConfig["schema"]; ok {
		for _, v := range schema.AsValueSlice() {
			s := v.AsValueMap()
			input = append(input, rest.ReferenceTableSchemaInput{
				Name: s["name"].AsString(),
				Type: s["type"].AsString(),
			})
		}
	}
	return input, diags

}

func referenceTableToResourceData(refTable *rest.ReferenceTable, dataset *gql.Dataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("oid", refTable.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("label", refTable.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", refTable.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset", oid.DatasetOid(refTable.DatasetId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("checksum", refTable.Checksum); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	schema := make([]map[string]interface{}, len(dataset.Typedef.Def.Fields))
	for i, field := range dataset.Typedef.Def.Fields {
		schema[i] = map[string]interface{}{
			"name": field.Name,
			"type": field.Type.Rep,
		}
	}
	if err := data.Set("schema", schema); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// TODO: add "primary_key" and "label_field" once API supports them in response.
	// Until then, we're unable to detect changes to those fields made outside of Terraform.

	return diags
}

func resourceReferenceTableCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newReferenceTableConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateReferenceTable(ctx, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create reference table",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceReferenceTableRead(ctx, data, meta)...)
}

func resourceReferenceTableRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetReferenceTable(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve reference table: %s", err.Error())
	}
	dataset, err := client.GetDataset(ctx, result.DatasetId)
	if err != nil {
		return diag.Errorf("failed to retrieve reference table dataset: %s", err.Error())
	}
	return referenceTableToResourceData(result, dataset, data)
}

func resourceReferenceTableUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	var err error

	// If only the source_file field changed, i.e. the file was moved or renamed, we can ignore it.
	// If the actual file contents have changed, the checksum would also be different.
	if !data.HasChangeExcept("source_file") {
		return nil
	}

	// If the file (i.e. the checksum) or schema have been modified, need to use the PUT method to
	// fully replace the reference table. Otherwise, we can use PATCH to only update the metadata.
	// TODO: remove primary_key and label_field below, API will support PATCHing them soon
	fieldsRequiringPut := []string{"checksum", "schema", "primary_key", "label_field"}
	if data.HasChanges(fieldsRequiringPut...) {
		config, diags := newReferenceTableConfig(data)
		if diags.HasError() {
			return diags
		}
		_, err = client.UpdateReferenceTable(ctx, data.Id(), config)
	} else {
		metadataConfig, diags := newReferenceTableMetadataConfig(data, true)
		if diags.HasError() {
			return diags
		}
		_, err = client.UpdateReferenceTableMetadata(ctx, data.Id(), metadataConfig)
	}

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update reference table [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return append(diags, resourceReferenceTableRead(ctx, data, meta)...)
}

func resourceReferenceTableDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteReferenceTable(ctx, data.Id()); err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			// reference table has already been deleted, ignore error
			return diags
		}
		return diag.Errorf("failed to delete reference table: %s", err)
	}
	return diags
}
