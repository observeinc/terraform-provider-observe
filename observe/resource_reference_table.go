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
		CustomizeDiff: rejectUploadOnlyChangesForInitialSourceFile,
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
				Optional:         true,
				ExactlyOneOf:     []string{"source_file", "initial_source_file"},
				RequiredWith:     []string{"checksum"},
				ValidateDiagFunc: validateFilePath(stringPtr(".csv")),
			},
			"initial_source_file": {
				Type:             schema.TypeString,
				Description:      descriptions.Get("reference_table", "schema", "initial_source_file"),
				Optional:         true,
				ExactlyOneOf:     []string{"source_file", "initial_source_file"},
				ValidateDiagFunc: validateFilePath(stringPtr(".csv")),
			},
			// checksum is used to avoid storing the entire file in the state and needing to fetch the
			// entire file every time to compare against for detecting changes.
			// See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_object
			"checksum": { // MD5 hash of source file contents
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"initial_source_file"},
				Description: descriptions.Get("reference_table", "schema", "checksum") +
					"Can be computed using `filemd5(\"<source_file>\")`. " +
					"Required with `source_file`, and must not be set with `initial_source_file`.",
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
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("reference_table", "schema", "schema", "name"),
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("reference_table", "schema", "schema", "type"),
						},
					},
				},
				Description: descriptions.Get("reference_table", "schema", "schema", "description"),
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
		SourceFilePath: uploadedSourceFilePath(data),
		Schema:         schemaInput,
		Metadata:       *metadataInput,
	}

	return input, diags
}

// uploadedSourceFilePath returns the file to upload: source_file, or
// initial_source_file, which is uploaded only when the table is created.
func uploadedSourceFilePath(data *schema.ResourceData) string {
	if path := data.Get("source_file").(string); path != "" {
		return path
	}
	return data.Get("initial_source_file").(string)
}

// rowsManagedOutsideTerraform reports whether the table was seeded from
// initial_source_file, so Terraform never uploads its rows after create.
func rowsManagedOutsideTerraform(data interface{ Get(string) interface{} }) bool {
	return data.Get("initial_source_file").(string) != ""
}

// rejectUploadOnlyChangesForInitialSourceFile fails the plan when a table
// seeded from initial_source_file changes schema or label_field. The API
// applies both only with a file upload, which would replace the rows edited in
// Observe, and replacing the table would lose them as well as its dataset id.
func rejectUploadOnlyChangesForInitialSourceFile(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if diff.Id() == "" || !rowsManagedOutsideTerraform(diff) {
		return nil
	}
	for _, field := range []string{"schema", "label_field"} {
		if diff.HasChange(field) {
			return fmt.Errorf("%s cannot be changed on a reference table created from initial_source_file, "+
				"because applying it requires uploading a file, which replaces the rows edited in Observe. "+
				"To change it, switch to source_file (with checksum) in the same apply: the file is uploaded "+
				"and replaces every row, and the table keeps its dataset id. Switch back to initial_source_file "+
				"afterwards to allow row edits in Observe again", field)
		}
	}
	return nil
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

	if err := data.Set("primary_key", refTable.PrimaryKey); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	labelField := ""
	if refTable.LabelField != nil {
		labelField = *refTable.LabelField
	}
	if err := data.Set("label_field", labelField); err != nil {
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

	// If the file (i.e. the checksum) or schema have been modified, need to use the PUT method to
	// fully replace the reference table. Otherwise, we can use PATCH to only update the metadata.
	// TODO: the API PATCHes primary_key, so remove it below; remove label_field
	// once the API PATCHes it too
	fieldsRequiringPut := []string{"checksum", "schema", "primary_key", "label_field"}
	uploadRequired := data.HasChanges(fieldsRequiringPut...)
	// Terraform never uploads rows seeded from initial_source_file: primary_key
	// goes through PATCH, and the plan rejects schema and label_field changes.
	if rowsManagedOutsideTerraform(data) {
		uploadRequired = false
	}

	// If only the source_file field changed, i.e. the file was moved or renamed, we can ignore it.
	// If the actual file contents have changed, the checksum would also be different.
	// initial_source_file is read only on create, and switching between the two
	// fields uploads only through a checksum change.
	if !uploadRequired && !data.HasChangesExcept("source_file", "initial_source_file") {
		return nil
	}

	if uploadRequired {
		config, configDiags := newReferenceTableConfig(data)
		if configDiags.HasError() {
			return configDiags
		}
		_, err = client.UpdateReferenceTable(ctx, data.Id(), config)
	} else {
		metadataConfig, metadataDiags := newReferenceTableMetadataConfig(data, true)
		if metadataDiags.HasError() {
			return metadataDiags
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
