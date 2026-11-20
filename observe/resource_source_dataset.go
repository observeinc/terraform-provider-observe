package observe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const (
	sourceDatasetStorageNative  = "Native"
	sourceDatasetStorageIceberg = "Iceberg"
	sourceDatasetSchemaTypeAny  = "Any"
	// SDKv2 cannot store null for an int, so -1 stands for "no retention set
	// on this dataset" in state and configuration.
	sourceDatasetRetentionInherit = -1
)

var (
	sourceDatasetStorages    = []string{sourceDatasetStorageNative, sourceDatasetStorageIceberg}
	sourceDatasetSchemaTypes = []string{sourceDatasetSchemaTypeAny, "OtelLogs", "OtelMetrics", "Prometheus", "K8sEntity", "OtelSpan"}
)

const legacySourceDatasetStateError = "this state was written by the removed legacy observe_source_dataset resource; " +
	"run `terraform state rm <address>` and remove it from configuration. The dataset it pointed to is unaffected."

func sourceDatasetDescription(key string) string {
	return descriptions.Get("source_dataset", "schema", key)
}

func listEnums(values []string, desc string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "`" + v + "`"
	}
	return fmt.Sprintf("%s Accepted values: %s", desc, strings.Join(quoted, ", "))
}

func resourceSourceDataset() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("source_dataset", "description"),
		CreateContext: resourceSourceDatasetCreate,
		ReadContext:   resourceSourceDatasetRead,
		UpdateContext: resourceSourceDatasetUpdate,
		DeleteContext: resourceSourceDatasetDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceSourceDatasetImport},
		CustomizeDiff: validateSourceDatasetPlan,
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Version: 0,
			Type:    cty.EmptyObject,
			Upgrade: upgradeLegacySourceDatasetState,
		}},
		Schema: map[string]*schema.Schema{
			"label": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      sourceDatasetDescription("label"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: sourceDatasetDescription("description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true, // the API assigns a default icon when none is sent
				Description: sourceDatasetDescription("icon_url"),
			},
			"object_tags": objectTagsSchemaFieldOptionalNoConflict(),
			"data_retention_days": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(sourceDatasetRetentionInherit)),
				Description:      sourceDatasetDescription("data_retention_days"),
			},
			"effective_data_retention_days": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: sourceDatasetDescription("effective_data_retention_days"),
			},
			"schema_type": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          sourceDatasetSchemaTypeAny,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice(sourceDatasetSchemaTypes, false),
				Description:      listEnums(sourceDatasetSchemaTypes, sourceDatasetDescription("schema_type")),
			},
			"storage": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          sourceDatasetStorageNative,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice(sourceDatasetStorages, false),
				Description:      listEnums(sourceDatasetStorages, sourceDatasetDescription("storage")),
			},
			"storage_integration": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeStorageIntegration),
				DiffSuppressFunc: diffSuppressOIDVersion,
				Description:      sourceDatasetDescription("storage_integration"),
			},
			"external_write": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: sourceDatasetDescription("external_write"),
			},
			"force_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: sourceDatasetDescription("force_destroy"),
			},
			"deletion_protection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: sourceDatasetDescription("deletion_protection"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

// The legacy resource pointed at a raw-table dataset that is not an ingest
// source dataset, so adopting it into the new resource would be wrong.
func upgradeLegacySourceDatasetState(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf(legacySourceDatasetStateError)
}

func sourceDatasetLabelPackage(label string) string {
	if i := strings.LastIndex(label, "/"); i >= 0 {
		return label[:i]
	}
	return ""
}

func validateSourceDatasetPlan(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	isIceberg := d.Get("storage").(string) == sourceDatasetStorageIceberg
	if d.NewValueKnown("storage") && d.NewValueKnown("storage_integration") {
		integration := d.Get("storage_integration").(string)
		if isIceberg && integration == "" {
			return fmt.Errorf("storage = %q requires storage_integration", sourceDatasetStorageIceberg)
		}
		if !isIceberg && integration != "" {
			return fmt.Errorf("storage_integration requires storage = %q", sourceDatasetStorageIceberg)
		}
	}
	if d.NewValueKnown("storage") && d.NewValueKnown("external_write") && !isIceberg && d.Get("external_write").(bool) {
		return fmt.Errorf("external_write = true requires storage = %q", sourceDatasetStorageIceberg)
	}
	if d.Id() != "" && d.HasChange("label") && d.NewValueKnown("label") {
		o, n := d.GetChange("label")
		oldPkg, newPkg := sourceDatasetLabelPackage(o.(string)), sourceDatasetLabelPackage(n.(string))
		if oldPkg != newPkg {
			return fmt.Errorf("the package of a source dataset label cannot change (%q to %q); only the name after the last \"/\" can be renamed. "+
				"Create a new observe_source_dataset to use a different package; replacing a source dataset deletes its data", oldPkg, newPkg)
		}
	}
	return nil
}

// rawConfigSet reports whether key is set in configuration, so that an
// explicit zero value is distinguishable from an unset attribute.
func rawConfigSet(data *schema.ResourceData, key string) bool {
	cfg := data.GetRawConfig()
	if cfg.IsNull() || !cfg.Type().IsObjectType() || !cfg.Type().HasAttribute(key) {
		_, ok := data.GetOkExists(key) //nolint:staticcheck // fallback when raw config is unavailable
		return ok
	}
	return !cfg.GetAttr(key).IsNull()
}

func sourceDatasetCreateRequest(data *schema.ResourceData) (*rest.SourceDatasetCreateRequest, error) {
	storage := data.Get("storage").(string)
	schemaType := data.Get("schema_type").(string)
	req := &rest.SourceDatasetCreateRequest{
		Label: data.Get("label").(string),
		DataSource: rest.SourceDatasetDataSourceInput{
			Kind:   "Ingest",
			Ingest: rest.SourceDatasetIngestInput{Storage: storage, SchemaType: schemaType},
		},
	}
	if v := data.Get("description").(string); v != "" {
		req.Description = &v
	}
	if v := data.Get("icon_url").(string); v != "" {
		req.Icon = &v
	}
	if tags := objectTagsMapFromReader(data); len(tags) > 0 {
		req.ObjectTags = tags
	}
	if days := data.Get("data_retention_days").(int); rawConfigSet(data, "data_retention_days") && days != sourceDatasetRetentionInherit {
		days64 := int64(days)
		req.DataRetentionDays = &days64
	}
	if storage == sourceDatasetStorageIceberg {
		siOID, err := oid.NewOID(data.Get("storage_integration").(string))
		if err != nil {
			return nil, fmt.Errorf("invalid storage_integration: %w", err)
		}
		req.DataSource.Ingest.Iceberg = &rest.SourceDatasetIcebergInput{
			StorageIntegrationId: siOID.Id,
			ExternalWrite:        data.Get("external_write").(bool),
		}
	}
	return req, nil
}

func sourceDatasetUpdateRequest(data *schema.ResourceData) *rest.SourceDatasetUpdateRequest {
	req := &rest.SourceDatasetUpdateRequest{}
	if data.HasChange("label") {
		req.Label = stringPtr(data.Get("label").(string))
	}
	if data.HasChange("description") {
		req.DescriptionSpecified = true
		if v := data.Get("description").(string); v != "" {
			req.Description = &v
		}
	}
	if data.HasChange("icon_url") {
		req.IconSpecified = true
		if v := data.Get("icon_url").(string); v != "" {
			req.Icon = &v
		}
	}
	if data.HasChange("object_tags") {
		tags := objectTagsMapFromReader(data)
		req.ObjectTags = &tags
	}
	// Removing data_retention_days from configuration never sends null:
	// retention can live on the shared ingest backing, and null would clear
	// it for every dataset on that backing. Only an explicit -1 sends null.
	if data.HasChange("data_retention_days") && rawConfigSet(data, "data_retention_days") {
		req.DataRetentionDaysSpecified = true
		if days := data.Get("data_retention_days").(int); days != sourceDatasetRetentionInherit {
			days64 := int64(days)
			req.DataRetentionDays = &days64
		}
	}
	return req
}

func sourceDatasetAPIError(action, id string, err error) diag.Diagnostics {
	if rest.HasStatusCode(err, http.StatusForbidden) && rest.HasErrorType(err, "feature_not_enabled") {
		return diag.Errorf("failed to %s source dataset %s: the source dataset API (/v1/source-datasets) is not enabled for this account: %s", action, id, err)
	}
	return diag.Errorf("failed to %s source dataset %s: %s", action, id, err)
}

func resourceSourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	req, err := sourceDatasetCreateRequest(data)
	if err != nil {
		return diag.FromErr(err)
	}
	ds, err := meta.(*observeclient.Client).Rest.CreateSourceDataset(ctx, req)
	if err != nil {
		return sourceDatasetAPIError("create", req.Label, err)
	}
	data.SetId(ds.Id)
	return resourceSourceDatasetRead(ctx, data, meta)
}

func resourceSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ds, err := meta.(*observeclient.Client).Rest.GetSourceDataset(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return sourceDatasetAPIError("read", data.Id(), err)
	}
	return sourceDatasetToResourceData(data, ds)
}

func sourceDatasetToResourceData(data *schema.ResourceData, ds *rest.SourceDatasetResource) (diags diag.Diagnostics) {
	set := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	// GET returns the label exactly as created, typed schema types included
	// (checked on testbox), so no suffix diff suppression is needed.
	set("label", ds.Label)
	description, icon := "", ""
	if ds.Description != nil {
		description = *ds.Description
	}
	if ds.Icon != nil {
		icon = *ds.Icon
	}
	set("description", description)
	set("icon_url", icon)
	tags := make(map[string]interface{}, len(ds.ObjectTags))
	for k, v := range ds.ObjectTags {
		tags[k] = encodeCSVValues(v)
	}
	set("object_tags", tags)
	if ds.DataRetentionDays != nil {
		set("data_retention_days", int(*ds.DataRetentionDays))
	} else {
		set("data_retention_days", sourceDatasetRetentionInherit)
	}
	set("effective_data_retention_days", int(ds.EffectiveDataRetentionDays))
	storage, schemaType, integration, externalWrite := sourceDatasetStorageNative, sourceDatasetSchemaTypeAny, "", false
	if ingest := ds.DataSource.Ingest; ingest != nil {
		if ingest.Storage != "" {
			storage = ingest.Storage
		}
		if ingest.SchemaType != "" {
			schemaType = ingest.SchemaType
		}
		if ingest.Iceberg != nil {
			externalWrite = ingest.Iceberg.ExternalWrite
			if ingest.Iceberg.StorageIntegration != nil && ingest.Iceberg.StorageIntegration.Id != "" {
				integration = oid.StorageIntegrationOid(ingest.Iceberg.StorageIntegration.Id).String()
			}
		}
	}
	set("storage", storage)
	set("schema_type", schemaType)
	set("storage_integration", integration)
	set("external_write", externalWrite)
	set("oid", oid.DatasetOid(ds.Id).String())
	return diags
}

func resourceSourceDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	req := sourceDatasetUpdateRequest(data)
	if *req == (rest.SourceDatasetUpdateRequest{}) {
		// Only provider-side attributes such as deletion_protection changed.
		return nil
	}
	if _, err := meta.(*observeclient.Client).Rest.UpdateSourceDataset(ctx, data.Id(), req); err != nil {
		return sourceDatasetAPIError("update", data.Id(), err)
	}
	return resourceSourceDatasetRead(ctx, data, meta)
}

// checkSourceDatasetDeletionAllowed runs on every delete Terraform performs:
// destroy, removal from configuration, and replacement (including -replace
// and taint). force_destroy overrides deletion_protection.
func checkSourceDatasetDeletionAllowed(data ResourceReader, id string) error {
	if data.Get("deletion_protection").(bool) && !data.Get("force_destroy").(bool) {
		return fmt.Errorf("cannot delete source dataset %s: deletion_protection is true. Deleting a source dataset deletes its data. "+
			"To delete or replace it, first apply deletion_protection = false, then run the destroy or replace again", id)
	}
	return nil
}

func resourceSourceDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := checkSourceDatasetDeletionAllowed(data, data.Id()); err != nil {
		return diag.FromErr(err)
	}
	err := meta.(*observeclient.Client).Rest.DeleteSourceDataset(ctx, data.Id(), data.Get("force_destroy").(bool))
	if err == nil || rest.HasStatusCode(err, http.StatusNotFound) {
		data.SetId("")
		return nil
	}
	if rest.HasStatusCode(err, http.StatusConflict) && rest.HasErrorType(err, "dependents_exist") {
		return diag.Errorf("failed to delete source dataset %s: %s. Set force_destroy = true to delete it anyway; datasets, dashboards and monitors that read from it stop working", data.Id(), err)
	}
	return sourceDatasetAPIError("delete", data.Id(), err)
}

// parseSourceDatasetImportID accepts a numeric dataset id or a dataset OID.
func parseSourceDatasetImportID(value string) (string, error) {
	if strings.HasPrefix(value, "o:") {
		id, err := oid.NewOID(value)
		if err != nil {
			return "", err
		}
		if id.Type != oid.TypeDataset {
			return "", fmt.Errorf("expected a dataset OID, got type %q", id.Type)
		}
		return id.Id, nil
	}
	if value == "" || strings.Trim(value, "0123456789") != "" {
		return "", fmt.Errorf("expected a numeric source dataset id or a dataset OID, got %q", value)
	}
	return value, nil
}

func resourceSourceDatasetImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id, err := parseSourceDatasetImportID(data.Id())
	if err != nil {
		return nil, err
	}
	if err := data.Set("force_destroy", false); err != nil {
		return nil, err
	}
	if err := data.Set("deletion_protection", true); err != nil {
		return nil, err
	}
	data.SetId(id)
	return []*schema.ResourceData{data}, nil
}
