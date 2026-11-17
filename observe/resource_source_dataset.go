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
			Type:    (&schema.Resource{Schema: legacySourceDatasetSchemaV0()}).CoreConfigSchema().ImpliedType(),
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
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
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
				Computed:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice(sourceDatasetSchemaTypes, false),
				Description:      listEnums(sourceDatasetSchemaTypes, sourceDatasetDescription("schema_type")),
			},
			"storage": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateStringInSlice(sourceDatasetStorages, false),
				Description:      listEnums(sourceDatasetStorages, sourceDatasetDescription("storage")),
			},
			"storage_integration": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true, // omitting it must not plan a data-deleting replace
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeStorageIntegration),
				DiffSuppressFunc: diffSuppressOIDVersion,
				Description:      sourceDatasetDescription("storage_integration"),
			},
			"external_write": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: sourceDatasetDescription("external_write"),
			},
			"force_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: sourceDatasetDescription("force_destroy"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

// legacySourceDatasetSchemaV0 is the schema of the removed observe_source_dataset
// resource, kept only to decode version 0 state for the upgrader.
func legacySourceDatasetSchemaV0() map[string]*schema.Schema {
	field := &schema.Resource{Schema: map[string]*schema.Schema{
		"name":          {Type: schema.TypeString, Required: true},
		"type":          {Type: schema.TypeString, Required: true},
		"sql_type":      {Type: schema.TypeString, Required: true},
		"is_enum":       {Type: schema.TypeBool, Optional: true, Default: false},
		"is_searchable": {Type: schema.TypeBool, Optional: true, Default: false},
		"is_hidden":     {Type: schema.TypeBool, Optional: true, Default: false},
		"is_const":      {Type: schema.TypeBool, Optional: true, Default: false},
		"is_metric":     {Type: schema.TypeBool, Optional: true, Default: false},
	}}
	return map[string]*schema.Schema{
		"workspace":                {Type: schema.TypeString, Optional: true, Computed: true},
		"name":                     {Type: schema.TypeString, Required: true},
		"schema":                   {Type: schema.TypeString, Required: true},
		"table_name":               {Type: schema.TypeString, Required: true},
		"source_update_table_name": {Type: schema.TypeString, Required: true},
		"valid_from_field":         {Type: schema.TypeString, Required: true},
		"batch_seq_field":          {Type: schema.TypeString, Optional: true},
		"is_insert_only":           {Type: schema.TypeBool, Optional: true, Default: false},
		"field":                    {Type: schema.TypeSet, Required: true, Elem: field},
		"description":              {Type: schema.TypeString, Optional: true},
		"icon_url":                 {Type: schema.TypeString, Optional: true},
		"oid":                      {Type: schema.TypeString, Computed: true},
		"freshness":                {Type: schema.TypeString, Optional: true},
	}
}

// The legacy resource pointed at a raw-table dataset that is not an ingest
// source dataset, so adopting it into the new resource would be wrong.
func upgradeLegacySourceDatasetState(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf(legacySourceDatasetStateError)
}

// effectiveSourceDatasetStorage returns the planned storage, falling back to
// the create default when it is unset and not yet known.
func effectiveSourceDatasetStorage(v string) string {
	if v == "" {
		return sourceDatasetStorageNative
	}
	return v
}

func sourceDatasetLabelPackage(label string) string {
	if i := strings.LastIndex(label, "/"); i >= 0 {
		return label[:i]
	}
	return ""
}

func validateSourceDatasetPlan(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	// The backing check runs on every plan of an existing resource, not only
	// when a config change forces a replace: -replace and replace_triggered_by
	// replace without any config diff, so they must be covered too.
	if err := checkSourceDatasetReplaceKeepsBacking(d); err != nil {
		return err
	}
	replacing := sourceDatasetBackingReplaced(d)
	storage, storageKnown := effectiveSourceDatasetValue(d, "storage", cty.StringVal(sourceDatasetStorageNative), replacing)
	integration, integrationKnown := effectiveSourceDatasetValue(d, "storage_integration", cty.StringVal(""), replacing)
	externalWrite, externalWriteKnown := effectiveSourceDatasetValue(d, "external_write", cty.False, replacing)
	if storageKnown && integrationKnown {
		isIceberg := storage.AsString() == sourceDatasetStorageIceberg
		if isIceberg && integration.AsString() == "" {
			return fmt.Errorf("storage = %q requires storage_integration", sourceDatasetStorageIceberg)
		}
		if !isIceberg && integration.AsString() != "" {
			return fmt.Errorf("storage_integration requires storage = %q", sourceDatasetStorageIceberg)
		}
	}
	if storageKnown && externalWriteKnown && storage.AsString() != sourceDatasetStorageIceberg && externalWrite.True() {
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

// sourceDatasetBackingForceNewKeys are the ForceNew attributes describing the
// backing; changing any of them replaces the source dataset.
var sourceDatasetBackingForceNewKeys = []string{"schema_type", "storage", "storage_integration", "external_write"}

// sourceDatasetBackingReplaced reports whether an existing resource will be
// replaced because a backing attribute set in configuration differs from state.
func sourceDatasetBackingReplaced(d *schema.ResourceDiff) bool {
	if d.Id() == "" {
		return false
	}
	cfg, state := d.GetRawConfig(), d.GetRawState()
	if cfg.IsNull() || !cfg.IsKnown() || state.IsNull() {
		return false
	}
	for _, key := range sourceDatasetBackingForceNewKeys {
		if !cfg.Type().HasAttribute(key) || !state.Type().HasAttribute(key) {
			continue
		}
		c, s := cfg.GetAttr(key), state.GetAttr(key)
		if !c.IsKnown() || c.IsNull() || !s.IsKnown() || s.IsNull() {
			continue
		}
		if key == "storage_integration" {
			if diffSuppressOIDVersion(key, s.AsString(), c.AsString(), nil) {
				continue
			}
		}
		if !c.RawEquals(s) {
			return true
		}
	}
	return false
}

// checkSourceDatasetReplaceKeepsBacking requires an existing Iceberg or
// external-write dataset to declare its backing in configuration. If the
// dataset is ever replaced, unset attributes take their create defaults, so
// the new dataset would silently be Native or lose external write. The check
// does not depend on whether this plan replaces: a forced replace (-replace,
// replace_triggered_by) has no config diff to detect. Taint skips
// CustomizeDiff entirely, so it is covered only by the configuration this
// check has already required on earlier plans.
func checkSourceDatasetReplaceKeepsBacking(d *schema.ResourceDiff) error {
	cfg, state := d.GetRawConfig(), d.GetRawState()
	if cfg.IsNull() || !cfg.IsKnown() || state.IsNull() {
		return nil
	}
	stateAttr := func(key string) cty.Value {
		if !state.Type().HasAttribute(key) {
			return cty.NilVal
		}
		v := state.GetAttr(key)
		if !v.IsKnown() || v.IsNull() {
			return cty.NilVal
		}
		return v
	}
	s, ew := stateAttr("storage"), stateAttr("external_write")
	iceberg := s != cty.NilVal && s.AsString() == sourceDatasetStorageIceberg
	externalWrite := ew != cty.NilVal && ew.True()
	if !iceberg && !externalWrite {
		return nil
	}
	// An explicit non-Iceberg storage is a deliberate move off Iceberg; the
	// integration and external write are then meant to be dropped.
	if cfg.Type().HasAttribute("storage") {
		if c := cfg.GetAttr("storage"); c.IsKnown() && !c.IsNull() && c.AsString() != sourceDatasetStorageIceberg {
			return nil
		}
	}
	isNull := func(key string) bool {
		return !cfg.Type().HasAttribute(key) || cfg.GetAttr(key).IsNull()
	}
	var missing []string
	if iceberg {
		for _, key := range []string{"storage", "storage_integration"} {
			if isNull(key) {
				missing = append(missing, key)
			}
		}
	}
	if externalWrite && isNull("external_write") {
		missing = append(missing, "external_write")
	}
	if len(missing) == 0 {
		return nil
	}
	storage := sourceDatasetStorageIceberg
	if s != cty.NilVal {
		storage = s.AsString()
	}
	var lost []string
	switch {
	case iceberg && isNull("storage"):
		lost = append(lost, "Native storage")
	case iceberg && isNull("storage_integration"):
		lost = append(lost, "no storage integration")
	}
	if externalWrite && isNull("external_write") {
		lost = append(lost, "external write off")
	}
	return fmt.Errorf("this source dataset is backed by storage = %q (external_write = %t), and %s unset in configuration. "+
		"If the dataset is replaced (a ForceNew change, -replace, replace_triggered_by or taint), unset attributes take their defaults, "+
		"so the new dataset would have %s and its data would be deleted. "+
		"Set %s explicitly in configuration",
		storage, externalWrite, joinIsAre(missing), strings.Join(lost, " and "), strings.Join(missing, ", "))
}

func joinIsAre(keys []string) string {
	if len(keys) == 1 {
		return keys[0] + " is"
	}
	return strings.Join(keys, ", ") + " are"
}

// effectiveSourceDatasetValue returns the value an attribute will have after
// apply: the configured value if set, else the value in state (unless the
// resource is being replaced), else the create default. Optional+Computed
// attributes that are unset in configuration are unknown in the plan of a new
// resource, so the plan value alone is not enough.
// The bool result is false when the configured value is not yet known.
func effectiveSourceDatasetValue(d *schema.ResourceDiff, key string, createDefault cty.Value, replacing bool) (cty.Value, bool) {
	cfg := d.GetRawConfig()
	if !cfg.IsNull() && cfg.IsKnown() && cfg.Type().IsObjectType() && cfg.Type().HasAttribute(key) {
		v := cfg.GetAttr(key)
		if !v.IsKnown() {
			return cty.NilVal, false
		}
		if !v.IsNull() {
			return v, true
		}
	}
	if d.Id() != "" && !replacing {
		state := d.GetRawState()
		if !state.IsNull() && state.Type().IsObjectType() && state.Type().HasAttribute(key) {
			if v := state.GetAttr(key); v.IsKnown() && !v.IsNull() {
				return v, true
			}
		}
	}
	return createDefault, true
}

// rawConfigSet reports whether key is set in configuration, so that an
// explicit zero value is distinguishable from an unset attribute.
func rawConfigSet(data sourceDatasetReader, key string) bool {
	cfg := data.GetRawConfig()
	if cfg.IsNull() || !cfg.Type().IsObjectType() || !cfg.Type().HasAttribute(key) {
		_, ok := data.GetOkExists(key) //nolint:staticcheck // fallback when raw config is unavailable
		return ok
	}
	return !cfg.GetAttr(key).IsNull()
}

func sourceDatasetCreateRequest(data *schema.ResourceData) (*rest.SourceDatasetCreateRequest, error) {
	storage := effectiveSourceDatasetStorage(data.Get("storage").(string))
	schemaType := data.Get("schema_type").(string)
	if schemaType == "" {
		schemaType = sourceDatasetSchemaTypeAny
	}
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
	if rawConfigSet(data, "data_retention_days") {
		days := int64(data.Get("data_retention_days").(int))
		req.DataRetentionDays = &days
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

// sourceDatasetReader is the subset of *schema.ResourceData used to build
// requests, so the update request can be unit tested with a fake.
type sourceDatasetReader interface {
	Get(string) interface{}
	GetOk(string) (interface{}, bool)
	GetOkExists(string) (interface{}, bool)
	HasChange(string) bool
	GetRawConfig() cty.Value
}

func sourceDatasetUpdateRequest(data sourceDatasetReader) *rest.SourceDatasetUpdateRequest {
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
	// it for every dataset on that backing.
	if data.HasChange("data_retention_days") && rawConfigSet(data, "data_retention_days") {
		days := int64(data.Get("data_retention_days").(int))
		req.DataRetentionDays = &days
		req.DataRetentionDaysSpecified = true
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
	}
	set("effective_data_retention_days", int(ds.EffectiveDataRetentionDays))
	storage, schemaType, integration, externalWrite := "", "", "", false
	if ingest := ds.DataSource.Ingest; ingest != nil {
		storage, schemaType = ingest.Storage, ingest.SchemaType
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
	if _, err := meta.(*observeclient.Client).Rest.UpdateSourceDataset(ctx, data.Id(), req); err != nil {
		return sourceDatasetAPIError("update", data.Id(), err)
	}
	return resourceSourceDatasetRead(ctx, data, meta)
}

func resourceSourceDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	data.SetId(id)
	return []*schema.ResourceData{data}, nil
}
