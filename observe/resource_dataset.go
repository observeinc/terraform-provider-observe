package observe

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	schemaDatasetWorkspaceDescription                     = "OID of workspace dataset is contained in."
	schemaDatasetNameDescription                          = "Dataset name. Must be unique within workspace."
	schemaDatasetDescriptionDescription                   = "Dataset description."
	schemaDatasetIconDescription                          = "Icon image."
	schemaDatasetPathCostDescription                      = "Path cost is used to weigh graph link computation."
	schemaDatasetOIDDescription                           = "The Observe ID for dataset."
	schemaDatasetFreshnessDescription                     = "Target freshness for dataset. This impacts how frequently the dataset query will be run."
	schemaDatasetOnDemandMaterializationLengthDescription = "The maximum on-demand materialization length for the dataset, in nanoseconds. " +
		"If unset, the default value in the transformer config will be used instead."
	schemaDatasetInputsDescription = "The inputs map binds dataset OIDs to labels which can be referenced within stage pipelines."

	schemaDatasetStageDescription = "Each stage processes an input according to the provided pipeline. " +
		"If no input is provided, a stage will implicitly follow on from the result of its predecessor."
	schemaDatasetStageAliasDescription = "The stage alias is the label by which subsequent stages can refer to the results of this stage."
	schemaDatasetStageInputDescription = "The stage input defines what input should be used as a starting point for the stage pipeline. " +
		"It must refer to a label contained in `inputs`, or a previous stage `alias`. " +
		"The stage input can be omitted if a dataset has a single input."
	schemaDatasetStagePipelineDescription = "An OPAL snippet defining a transformation on the selected input."
)

func resourceDataset() *schema.Resource {
	return &schema.Resource{
		Description:   "An description",
		CreateContext: resourceDatasetCreate,
		ReadContext:   resourceDatasetRead,
		UpdateContext: resourceDatasetUpdate,
		DeleteContext: resourceDatasetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if datasetRecomputeOID(d) {
				return d.SetNewComputed("oid")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaDatasetWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDatasetNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDatasetDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDatasetIconDescription,
			},
			"path_cost": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: schemaDatasetPathCostDescription,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDatasetOIDDescription,
			},
			"freshness": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      schemaDatasetFreshnessDescription,
			},
			"on_demand_materialization_length": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      schemaDatasetOnDemandMaterializationLengthDescription,
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
				Description:      schemaDatasetInputsDescription,
			},
			"stage": {
				Type:        schema.TypeList,
				MinItems:    1,
				Required:    true,
				Description: schemaDatasetStageDescription,
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
							Description: schemaDatasetStageAliasDescription,
						},
						"input": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: schemaDatasetStageInputDescription,
						},
						"pipeline": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      schemaDatasetStagePipelineDescription,
							DiffSuppressFunc: diffSuppressPipeline,
						},
					},
				},
			},
		},
	}
}

func newDatasetConfig(data *schema.ResourceData) (*gql.DatasetInput, *gql.MultiStageQueryInput, diag.Diagnostics) {
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

	if v, ok := data.GetOk("path_cost"); ok {
		input.PathCost = types.Int64Scalar(v.(int)).Ptr()
	} else {
		input.PathCost = types.Int64Scalar(0).Ptr()
	}

	return input, query, diags
}

func datasetToResourceData(d *gql.Dataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Label); err != nil {
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

	if d.PathCost != nil {
		if err := data.Set("path_cost", d.PathCost); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if diags.HasError() {
		return diags
	}

	if d.Transform != nil && d.Transform.Current != nil {
		if err := flattenAndSetQuery(data, d.Transform.Current.Query.Stages); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenAndSetQuery(data *schema.ResourceData, gqlstages []*gql.StageQuery) error {
	if len(gqlstages) == 0 {
		return nil
	}

	queryData, err := flattenQuery(gqlstages)
	if err != nil {
		return err
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
		return err
	}

	stages := make([]interface{}, len(queryData.Stages))
	for i, stage := range queryData.Stages {
		s := map[string]interface{}{
			"pipeline": stage.Pipeline,
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
		return err
	}

	return nil
}

func resourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	input, queryInput, diags := newDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	wsid, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.SaveDataset(ctx, wsid.Id, input, queryInput)
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

	result, err := client.SaveDataset(ctx, wsid.Id, input, queryInput)
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
