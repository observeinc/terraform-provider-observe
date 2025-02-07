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
			"source": {
				Type:             schema.TypeString,
				Description:      descriptions.Get("reference_table", "schema", "source"),
				Required:         true,
				ValidateDiagFunc: validateFilePath(stringPtr(".csv")),
			},
			// checksum is used to avoid storing the entire file in the state and needing to fetch the
			// entire file every time to compare against for detecting changes.
			// See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_object
			"checksum": { // MD5 hash of source file contents
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("reference_table", "schema", "checksum"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("reference_table", "schema", "description"),
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
			// TODO: support "schema"
		},
	}
}

func newReferenceTableConfig(data *schema.ResourceData) (input *rest.ReferenceTableInput, diags diag.Diagnostics) {
	metadataInput, diags := newReferenceTableMetadataConfig(data, false)
	if diags.HasError() {
		return nil, diags
	}

	input = &rest.ReferenceTableInput{
		Metadata:       *metadataInput,
		SourceFilePath: data.Get("source").(string),
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

func referenceTableToResourceData(d *rest.ReferenceTable, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("label", d.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", d.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("dataset", oid.DatasetOid(d.DatasetId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("checksum", d.Checksum); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// TODO: add "primary_key" and "label_field" once supported

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
	return referenceTableToResourceData(result, data)
}

func resourceReferenceTableUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	var err error

	// If only the source field changed, i.e. the file was moved or renamed, we can ignore it.
	// If the actual file contents have changed, the checksum would also be different.
	if !data.HasChangeExcept("source") {
		return nil
	}

	// If the file has been modified (i.e. the checksum), need to use the PUT method to fully
	// replace the reference table. Otherwise, we can use PATCH to only update the metadata.
	if data.HasChanges("checksum", "label_field") { // TODO: remove label_field here once PATCH supported
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
		return diag.Errorf("failed to delete reference table: %s", err)
	}
	return diags
}
