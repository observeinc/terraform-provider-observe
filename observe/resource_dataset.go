package observe

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// Terraform-level options for rematerialization mode. This is because Terraform exposes
// some options the API doesn't have and we shouldn't mix them up
type TerraformRematerializationMode string

const (
	RematerializationModeRematerialize             = TerraformRematerializationMode(gql.RematerializationModeRematerialize)
	RematerializationModeSkipRematerialization     = TerraformRematerializationMode(gql.RematerializationModeSkiprematerialization)
	RematerializationModeMustSkipRematerialization = TerraformRematerializationMode("MustSkipRematerialization")
)

var AllRematerializationModes = []TerraformRematerializationMode{
	RematerializationModeRematerialize,
	RematerializationModeSkipRematerialization,
	RematerializationModeMustSkipRematerialization,
}

func resourceDataset() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("dataset", "description"),
		CreateContext: resourceDatasetCreate,
		ReadContext:   resourceDatasetRead,
		UpdateContext: resourceDatasetUpdate,
		DeleteContext: resourceDatasetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceDatasetCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      descriptions.Get("dataset", "schema", "name"),
				ValidateDiagFunc: validateDatasetName(),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"path_cost": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "path_cost"),
			},
			"on_demand_materialization_length": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("dataset", "schema", "on_demand_materialization_length"),
			},
			"freshness": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("transform", "schema", "freshness"),
			},
			"acceleration_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("dataset", "schema", "acceleration_disabled"),
			},
			"acceleration_disabled_source": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateEnums(gql.AllAccelerationDisabledSource),
				Description:      descriptions.Get("dataset", "schema", "acceleration_disabled_source"),
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
				Description:      descriptions.Get("transform", "schema", "inputs"),
			},
			"data_table_view_state": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      descriptions.Get("dataset", "schema", "data_table_view_state"),
			},
			"storage_integration": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeStorageIntegration),
				Description:      descriptions.Get("dataset", "schema", "storage_integration"),
			},
			"stage": {
				Type:        schema.TypeList,
				MinItems:    1,
				Required:    true,
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:     schema.TypeString,
							Optional: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// ignore alias for last stage, because it won't be set anyway
								stage := d.Get("stage").([]interface{})
								return k == fmt.Sprintf("stage.%d.alias", len(stage)-1)
							},
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": {
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: diffSuppressPipeline,
							Description:      descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"output_stage": {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
			"rematerialization_mode": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateEnums(AllRematerializationModes),
				Description:      descriptions.Get("dataset", "schema", "rematerialization_mode"),
			},
		},
	}
}

// ResourceReader is satisfied by both schema.ResourceData and schema.ResourceDiff
// (necessary so we can call newDatasetConfig from CustomizeDiff, which uses a schema.ResourceDiff)
type ResourceReader interface {
	Get(key string) interface{}
	GetOk(key string) (interface{}, bool)
}

func resourceDatasetCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	client := meta.(*observe.Client)

	if datasetRecomputeOID(d) {
		err := d.SetNewComputed("oid")
		if err != nil {
			return err
		}
	}
	if err := validateDatasetChanges(ctx, d, client); err != nil {
		return err
	}
	return nil
}

func validateDatasetChanges(ctx context.Context, d *schema.ResourceDiff, client *observe.Client) error {
	// Fields that could use server-side validation ("name" because we enforce uniqueness)
	if d.HasChange("inputs") || d.HasChange("stage") || d.HasChange("name") {
		// We first need to check if the resource is fully known. For example, if inputs is
		// referencing a dataset that's being created in the same terraform run, we don't know its
		// ID during the plan stage and therefore can't do a dry-run save.
		// If a value is not known, .Get() will return the zero value, so we could perhaps get
		// away with not knowing the value for certain optional fields that wouldn't affect the
		// validation. But if we get it wrong, it could prevent a valid dataset create/update.
		// So for now, if all fields aren't fully known, we skip the dry-run validation.
		if d.GetRawConfig().IsWhollyKnown() {
			wsid, _ := oid.NewOID(d.Get("workspace").(string))
			input, queryInput, diags := newDatasetConfig(d)
			if diags.HasError() {
				return fmt.Errorf("invalid dataset config: %s", concatenateDiagnosticsToStr(diags))
			}
			if id := d.Id(); id != "" {
				input.Id = &id
			}

			result, err := client.SaveDatasetDryRun(ctx, wsid.Id, input, queryInput)
			if err != nil {
				// Ignore timeout errors. TODO: no longer necessary once the backend is updated.
				if errors.Is(err, context.DeadlineExceeded) {
					return nil
				} else {
					return fmt.Errorf("dataset save dry-run failed: %s", err.Error())
				}
			}

			// Ideally in addition to erroring for "must_skip_rematerialization", we'd also emit warnings
			// for "skip_rematerialization". But terraform doesn't let us do that here.
			rematerializationMode := getRematerializationMode(client, d)
			if rematerializationMode == RematerializationModeMustSkipRematerialization && len(result.DematerializedDatasets) > 0 {
				return errors.New(rematerializationErrorStr(result.DematerializedDatasets))
			}

			// We could also check result.ErrorDatasets here for any downstream errors. But there
			// may be cases when a downstream dataset must be temporarily broken in order to make
			// certain changes one dataset at a time. So not erroring here to allow such changes.
			// Unfortunately, terraform won't let us emit a warning here. In the future, may
			// consider erroring in such cases by default and having some field/flag to ignore them.
		}
	}
	return nil
}

func newDatasetConfig(data ResourceReader) (*gql.DatasetInput, *gql.MultiStageQueryInput, diag.Diagnostics) {
	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, nil, diags
	}

	if query == nil {
		return nil, nil, diag.Errorf("no query provided")
	}

	overwriteSource := true
	input := &gql.DatasetInput{
		OverwriteSource: &overwriteSource,
	}

	if v, ok := data.GetOk("name"); ok {
		input.Label = v.(string)
	} else {
		return nil, nil, diag.Errorf("name not set")
	}

	if v, ok := data.GetOk("freshness"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		input.FreshnessDesired = types.Int64Scalar(t).Ptr()
	}

	if v, ok := data.GetOk("on_demand_materialization_length"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		input.OnDemandMaterializationLength = types.Int64Scalar(t).Ptr()
	}

	{
		// always reset to empty string if description not set
		input.Description = stringPtr(data.Get("description").(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	b := data.Get("acceleration_disabled").(bool)
	input.AccelerationDisabled = &b

	if v, ok := data.GetOk("acceleration_disabled_source"); ok {
		c := gql.AccelerationDisabledSource(toCamel(v.(string)))
		input.AccelerationDisabledSource = &c
	}

	if v, ok := data.GetOk("path_cost"); ok {
		input.PathCost = types.Int64Scalar(v.(int)).Ptr()
	} else {
		input.PathCost = types.Int64Scalar(0).Ptr()
		// null it is
	}

	if v, ok := data.GetOk("data_table_view_state"); ok {
		input.DataTableViewState = types.JsonObject(v.(string)).Ptr()
	} else {
		input.DataTableViewState = types.JsonObject("null").Ptr()
	}

	if v, ok := data.GetOk("storage_integration"); ok {
		oidVal, _ := oid.NewOID(v.(string))
		input.StorageIntegrationId = stringPtr(oidVal.Id)

	}

	return input, query, diags
}

func datasetToResourceData(d *gql.Dataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.FreshnessDesired != nil {
		if err := data.Set("freshness", d.FreshnessDesired.Duration().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.OnDemandMaterializationLength != nil {
		if err := data.Set("on_demand_materialization_length", d.OnDemandMaterializationLength.Duration().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Description != nil {
		if err := data.Set("description", d.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("acceleration_disabled", d.AccelerationDisabled); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("acceleration_disabled_source", toSnake(string(d.AccelerationDisabledSource))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	var currentCost int
	if v, ok := data.GetOk("path_cost"); ok {
		currentCost = v.(int)
	}

	if d.PathCost != nil && *d.PathCost.IntPtr() != currentCost {
		if err := data.Set("path_cost", d.PathCost); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.DataTableViewState != nil {
		if err := data.Set("data_table_view_state", d.DataTableViewState.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.StorageIntegrationId != nil {
		oid := oid.StorageIntegrationOid(*d.StorageIntegrationId)
		if err := data.Set("storage_integration", oid.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if diags.HasError() {
		return diags
	}

	if d.Transform != nil && d.Transform.Current != nil {
		_, err := flattenAndSetQuery(data, d.Transform.Current.Query.Stages, d.Transform.Current.Query.OutputStage)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenAndSetQuery(data *schema.ResourceData, gqlstages []gql.StageQuery, outputStage string) ([]string, error) {
	if len(gqlstages) == 0 {
		return make([]string, 0), nil
	}

	queryData, err := flattenQuery(gqlstages, outputStage)
	if err != nil {
		return nil, err
	}

	inputs := make(map[string]interface{}, 0)
	for name, input := range queryData.Inputs {
		id := oid.OID{
			Type: oid.TypeDataset,
			Id:   *input.Dataset,
		}

		// check for existing version timestamp we can maintain
		if v, ok := data.GetOk(fmt.Sprintf("inputs.%s", name)); ok {
			prv, err := oid.NewOID(v.(string))
			if err == nil && id.Id == prv.Id {
				id.Version = prv.Version
			}
		}
		inputs[name] = id.String()
	}

	if err := data.Set("inputs", inputs); err != nil {
		return nil, err
	}

	stages := make([]interface{}, len(queryData.Stages))
	for i, stage := range queryData.Stages {
		s := map[string]interface{}{
			"pipeline":     stage.Pipeline,
			"output_stage": stage.OutputStage,
		}
		if stage.Alias != nil {
			s["alias"] = stage.Alias
		}
		if stage.Input != nil {
			s["input"] = stage.Input
		} else if i == 0 {
			s["input"] = data.Get("stage.0.input")
		}
		stages[i] = s
	}

	if err := data.Set("stage", stages); err != nil {
		return nil, err
	}

	return queryData.StageIds, nil
}

func resourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	input, queryInput, diags := newDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	dependencyHandling := gql.DefaultDependencyHandling()
	if mode, ok := data.GetOk("rematerialization_mode"); ok {
		rematerializationMode := gql.RematerializationMode(toCamel(mode.(string)))
		dependencyHandling.RematerializationMode = &rematerializationMode

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "rematerialization_mode on a new dataset is a no-op",
		})
	}

	wsid, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.SaveDataset(ctx, wsid.Id, input, queryInput, dependencyHandling)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dataset",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceDatasetRead(ctx, data, meta)...)
}

func resourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDataset(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dataset [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return datasetToResourceData(result, data)
}

func resourceDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	input, queryInput, diags := newDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	id := data.Id()
	input.Id = &id
	wsid, _ := oid.NewOID(data.Get("workspace").(string))

	// If must_skip_rematerialization is set, do a dry-run to ensure it skips rematerialization.
	// We already do this in CustomizeDiff, but sometimes the plan is run beforehand (e.g. when a PR is created)
	// and the apply (using that saved plan) is run much later (e.g. when the PR is merged).
	// Something could have changed in the environment between them resulting in new dematerializations.
	rematerializationMode := getRematerializationMode(client, data)
	if rematerializationMode == RematerializationModeMustSkipRematerialization {
		if result, err := client.SaveDatasetDryRun(ctx, wsid.Id, input, queryInput); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("failed to update dataset [id=%s]", data.Id()),
				Detail:   err.Error(),
			})
			return diags
		} else if len(result.DematerializedDatasets) > 0 {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("failed to update dataset [id=%s]", data.Id()),
				Detail:   rematerializationErrorStr(result.DematerializedDatasets),
			})
			return diags
		}
	}

	dependencyHandling := gql.DefaultDependencyHandling()
	// Map the Terraform version of must_skip_rematerialization to GQL (do this
	// because must_skip_rematerialization doesn't exist at the API level)
	// Default dependency handling results in rematerialization, don't need to
	// map that case.
	switch rematerializationMode {
	case RematerializationModeSkipRematerialization, RematerializationModeMustSkipRematerialization:
		mode := gql.RematerializationModeSkiprematerialization
		dependencyHandling.RematerializationMode = &mode
	}

	result, err := client.SaveDataset(ctx, wsid.Id, input, queryInput, dependencyHandling)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update dataset [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return datasetToResourceData(result, data)
}

func resourceDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDataset(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dataset: %s", err)
	}
	return diags
}

func diffSuppressVersion(k, old, new string, d *schema.ResourceData) bool {
	if old == new {
		return true
	}

	if old == "" {
		return false
	}

	oldOID, err := oid.NewOID(old)
	if err != nil {
		log.Printf("[WARN] could not convert old %s %q to OID: %s\n", k, old, err)
		return false
	}

	newOID, err := oid.NewOID(new)
	if err != nil {
		log.Printf("[WARN] could not convert new %s %q to OID: %s\n", k, new, err)
		return false
	}

	// ignore version
	return oldOID.Type == newOID.Type && oldOID.Id == newOID.Id
}

func rematerializationErrorStr(dematerializedDatasets []gql.DatasetMaterialization) string {
	if len(dematerializedDatasets) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("The following dataset(s) will be rematerialized: ")
	for idx, dematerializedDataset := range dematerializedDatasets {
		if idx > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s (%s)", dematerializedDataset.GetDataset().Id, dematerializedDataset.GetDataset().Name)
	}
	sb.WriteString(`. If rematerialization is acceptable, remove rematerialization_mode and try again`)
	return sb.String()
}

func getRematerializationMode(client *observe.Client, data ResourceReader) TerraformRematerializationMode {
	rematerializationMode := RematerializationModeRematerialize
	if client.DefaultRematerializationMode != nil {
		rematerializationMode = TerraformRematerializationMode(toCamel(*client.DefaultRematerializationMode))
	}
	if mode, ok := data.GetOk("rematerialization_mode"); ok {
		rematerializationMode = TerraformRematerializationMode(toCamel(mode.(string)))
	}
	return rematerializationMode
}
