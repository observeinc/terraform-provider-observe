package observe

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const (
	correlationTagNameKey    = "name"
	correlationTagDatasetKey = "dataset"
	correlationTagColumnKey  = "column"
	correlationTagPathKey    = "path"
)

func resourceCorrelationTag() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("correlation_tag", "description"),
		CreateContext: resourceCorrelationTagCreate,
		ReadContext:   resourceCorrelationTagRead,
		UpdateContext: resourceCorrelationTagUpdate,
		DeleteContext: resourceCorrelationTagDelete,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			// If the dataset ID has changed, we need to recreate the resource. See comments on ForceNew below.
			if d.HasChange(correlationTagDatasetKey) {
				// Ignore version changes
				oldVal, newVal := d.GetChange(correlationTagDatasetKey)
				oldOid, oldErr := oid.NewOID(oldVal.(string))
				newOid, newErr := oid.NewOID(newVal.(string))
				if oldErr == nil && newErr == nil && oldOid.Id != newOid.Id {
					d.ForceNew(correlationTagDatasetKey)
				}
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			correlationTagDatasetKey: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				Description:      descriptions.Get("correlation_tag", "schema", correlationTagDatasetKey),
				DiffSuppressFunc: diffSuppressOIDVersion,
				// We should be enforcing ForceNew here, but unfortunately because the dataset oids contain a version,
				// the value of the oid is always "(known after apply)" when a dataset is updated. This means
				// that even though we're diff suppressing the version, the correlation tag will always
				// be recreated when the dataset is updated, because terraform can't guarantee the new value of
				// the oid during the plan stage. Instead doing it in the CustomizeDiff above, which does not
				// consider "(known after apply)" changes.
				// ForceNew:         true,
			},
			correlationTagNameKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("correlation_tag", "schema", correlationTagNameKey),
				ForceNew:    true,
			},
			correlationTagColumnKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("correlation_tag", "schema", correlationTagColumnKey),
				ForceNew:    true,
			},
			correlationTagPathKey: {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions.Get("correlation_tag", "schema", correlationTagPathKey),
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}

func newCorrelationTagConfig(data *schema.ResourceData) (params correlationTagParameters, diags diag.Diagnostics) {
	datasetOid, _ := oid.NewOID(data.Get(correlationTagDatasetKey).(string))
	dataset := datasetOid.Id

	tag, _ := data.Get(correlationTagNameKey).(string)
	column, _ := data.Get(correlationTagColumnKey).(string)
	objectPath, _ := data.Get(correlationTagPathKey).(string)
	path := gql.LinkFieldInput{
		Path:   &objectPath,
		Column: column,
	}
	params = correlationTagParameters{
		Dataset: dataset,
		Tag:     tag,
		Path:    path,
	}
	return
}

func constructCorrelationTagId(dataset, tag string, path gql.LinkFieldInput) string {
	// While we want to be able to configure correlation tags separately from dataset, these tags don't have an id in the backend.
	// Terraform doesn't like that. So, make up a tag that lets us retrieve (dataset, path, tag) combination from it.
	id, _ := json.Marshal(correlationTagParameters{
		Dataset: dataset,
		Tag:     tag,
		Path:    path,
	})
	return string(id)
}

func deconstructCorrelationTagId(id string) (params correlationTagParameters, err error) {
	err = json.Unmarshal([]byte(id), &params)
	if err != nil {
		return
	}
	return params, nil
}

func resourceCorrelationTagCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	params, diags := newCorrelationTagConfig(data)
	if diags.HasError() {
		return diags
	}

	// A correlation tag isn't a first-class object with its own identity -- it's
	// just a (dataset, tag, column, path) combination, and that combination is
	// its entire state. If it already exists (e.g. someone created it in the UI,
	// or in another `apply`), there's nothing left to reconcile, so adopt it into
	// this resource instead of failing.
	isPresent, err := client.IsCorrelationTagPresent(ctx, params.Dataset, params.Tag, params.Path)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to check for existing correlation tag",
			Detail:   err.Error(),
		})
	}

	if !isPresent {
		if err := client.CreateCorrelationTag(ctx, params.Dataset, params.Tag, params.Path); err != nil {
			return append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "failed to create correlation tag",
				Detail:   err.Error(),
			})
		}
	}

	data.SetId(constructCorrelationTagId(params.Dataset, params.Tag, params.Path))
	return append(diags, resourceCorrelationTagRead(ctx, data, meta)...)
}

func resourceCorrelationTagRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	cTagParams, err := deconstructCorrelationTagId(data.Id())
	if err != nil {
		return diag.Errorf("failed to deconstruct correlation tag id: %s", err.Error())
	}
	isPresent, err := client.IsCorrelationTagPresent(ctx, cTagParams.Dataset, cTagParams.Tag, cTagParams.Path)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	} else if !isPresent {
		// Mark the correlation tag as deleted.
		data.SetId("")
	}
	return diags
}

// resourceCorrelationTagUpdate is a no-op: correlation tags are immutable on the backend, so the
// only way to change them is to recreate. All fields except dataset are ForceNew; dataset changes
// are handled by CustomizeDiff (ForceNew when the dataset ID changes, suppressed when only the
// OID version changes). This function exists solely to satisfy the Terraform SDK requirement that
// resources with non-ForceNew fields define an Update.
func resourceCorrelationTagUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return resourceCorrelationTagRead(ctx, data, meta)
}

func resourceCorrelationTagDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	cTagParams, err := deconstructCorrelationTagId(data.Id())
	if err != nil {
		return diag.Errorf("failed to deconstruct correlation tag id: %s", err.Error())
	}
	err = client.DeleteCorrelationTag(ctx, cTagParams.Dataset, cTagParams.Tag, cTagParams.Path)
	if err != nil {
		return diag.Errorf("failed to delete correlation tag: %s", err.Error())
	}
	return diags
}

type correlationTagParameters struct {
	Dataset string
	Tag     string
	Path    gql.LinkFieldInput
}
