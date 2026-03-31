package observe

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const ldmDefaultInputName = "input"
const ldmDefaultStageID = "stage-0"

func resourceLogDerivedMetricDataset() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("log_derived_metric_dataset", "description"),
		CreateContext: resourceLogDerivedMetricDatasetCreate,
		ReadContext:   resourceLogDerivedMetricDatasetRead,
		UpdateContext: resourceLogDerivedMetricDatasetUpdate,
		DeleteContext: resourceLogDerivedMetricDatasetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceLogDerivedMetricDatasetCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
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
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "name"),
				ValidateDiagFunc: validateDatasetName(),
			},
			"metric_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "metric_name"),
			},
			"metric_type": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateEnums(gql.AllMetricTypes),
				DiffSuppressFunc: diffSuppressEnums,
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "metric_type"),
			},
			"unit": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "unit"),
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "interval"),
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
			"input": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(),
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "input"),
			},
			"query": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "",
				DiffSuppressFunc: diffSuppressPipeline,
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "query"),
			},
			"aggregation": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "aggregation", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateEnums(gql.AllLogDerivedMetricAggregationFunctions),
							DiffSuppressFunc: diffSuppressEnums,
							Description:      descriptions.Get("log_derived_metric_dataset", "schema", "aggregation", "function"),
						},
						"field_path": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: descriptions.Get("log_derived_metric_dataset", "schema", "aggregation", "field_path", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"column": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("log_derived_metric_dataset", "schema", "aggregation", "field_path", "column"),
									},
									"path": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "",
										Description: descriptions.Get("log_derived_metric_dataset", "schema", "aggregation", "field_path", "path"),
									},
								},
							},
						},
					},
				},
			},
			"metric_tag": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "metric_tag", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("log_derived_metric_dataset", "schema", "metric_tag", "name"),
						},
						"column": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("log_derived_metric_dataset", "schema", "metric_tag", "column"),
						},
						"path": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: descriptions.Get("log_derived_metric_dataset", "schema", "metric_tag", "path"),
						},
					},
				},
			},
		},
	}
}

func resourceLogDerivedMetricDatasetCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	client := meta.(*observe.Client)

	if datasetRecomputeOID(d) {
		if err := d.SetNewComputed("oid"); err != nil {
			return err
		}
	}

	return validateLogDerivedMetricDatasetChanges(ctx, d, client)
}

func validateLogDerivedMetricDatasetChanges(ctx context.Context, d *schema.ResourceDiff, client *observe.Client) error {
	if client.SkipDatasetDryRuns {
		return nil
	}

	if !(d.HasChange("input") || d.HasChange("query") || d.HasChange("aggregation") || d.HasChange("metric_name") ||
		d.HasChange("metric_type") || d.HasChange("unit") || d.HasChange("interval") ||
		d.HasChange("metric_tag") || d.HasChange("name")) {
		return nil
	}

	if !d.GetRawConfig().IsWhollyKnown() {
		return nil
	}

	wsid, _ := oid.NewOID(d.Get("workspace").(string))
	input, queryInput, logInput, diags := newLogDerivedMetricDatasetConfig(d)
	if diags.HasError() {
		return fmt.Errorf("invalid log-derived metric dataset config: %s", concatenateDiagnosticsToStr(diags))
	}
	if id := d.Id(); id != "" {
		input.Id = &id
	}

	_, err := client.SaveLogDerivedMetricDatasetDryRun(ctx, wsid.Id, input, queryInput, logInput)
	if err != nil {
		return fmt.Errorf("log-derived metric dataset save dry-run failed: %s", err.Error())
	}

	return nil
}

func newLDMShapingStageQueryInput(data ResourceReader) (gql.StageQueryInput, diag.Diagnostics) {
	inputOIDStr := data.Get("input").(string)
	parsedOID, err := oid.NewOID(inputOIDStr)
	if err != nil {
		return gql.StageQueryInput{}, diag.FromErr(fmt.Errorf("input: %w", err))
	}
	datasetID := parsedOID.Id
	if datasetID == "" {
		return gql.StageQueryInput{}, diag.Errorf("input: empty dataset id")
	}
	if _, err := strconv.ParseInt(datasetID, 10, 64); err != nil {
		return gql.StageQueryInput{}, diag.FromErr(fmt.Errorf("input: %w", errObjectIDInvalid))
	}

	pipeline := data.Get("query").(string)
	stageID := ldmDefaultStageID

	return gql.StageQueryInput{
		Id:       &stageID,
		Pipeline: pipeline,
		Input: []gql.InputDefinitionInput{
			{
				InputName: ldmDefaultInputName,
				DatasetId: &datasetID,
			},
		},
	}, nil
}

func newLogDerivedMetricDefinitionInput(data ResourceReader) (*gql.LogDerivedMetricDefinitionInput, diag.Diagnostics) {
	shaping, diags := newLDMShapingStageQueryInput(data)
	if diags.HasError() {
		return nil, diags
	}

	metricName := data.Get("metric_name").(string)

	aggRaw := data.Get("aggregation").([]interface{})
	if len(aggRaw) != 1 {
		return nil, diag.Errorf("aggregation must have exactly one block")
	}
	agg := aggRaw[0].(map[string]interface{})
	fnStr := agg["function"].(string)
	fn := gql.LogDerivedMetricAggregationFunction(toCamel(fnStr))

	aggIn := gql.LogDerivedMetricAggregationInput{
		Config: gql.LogDerivedMetricAggregationConfigInput{
			Function: fn,
		},
	}

	if fpRaw, ok := agg["field_path"].([]interface{}); ok && len(fpRaw) == 1 {
		fp := fpRaw[0].(map[string]interface{})
		aggIn.FieldPath = &gql.MetricTagPathInput{
			Column: fp["column"].(string),
			Path:   fp["path"].(string),
		}
	}

	tags := make([]gql.LogMetricTagInput, 0)
	if raw, ok := data.GetOk("metric_tag"); ok {
		for _, item := range raw.([]interface{}) {
			t := item.(map[string]interface{})
			tags = append(tags, gql.LogMetricTagInput{
				Name: t["name"].(string),
				FieldPath: gql.MetricTagPathInput{
					Column: t["column"].(string),
					Path:   t["path"].(string),
				},
			})
		}
	}

	out := &gql.LogDerivedMetricDefinitionInput{
		MetricName:   metricName,
		ShapingQuery: shaping,
		Aggregation:  aggIn,
		MetricTags:   tags,
	}

	if v, ok := data.GetOk("metric_type"); ok {
		// we expect the metric to be in snake_case, so we convert it to camelCase with lowercase first letter
		mt := gql.MetricType(toCamelLower(v.(string)))
		out.MetricType = &mt
	}
	if v, ok := data.GetOk("unit"); ok {
		out.Unit = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("interval"); ok {
		t, _ := time.ParseDuration(v.(string))
		dur := types.DurationScalar(t)
		out.Interval = &dur
	}

	return out, nil
}

func newLogDerivedMetricDatasetConfig(data ResourceReader) (*gql.DatasetInput, *gql.MultiStageQueryInput, *gql.LogDerivedMetricDefinitionInput, diag.Diagnostics) {
	logDef, diags := newLogDerivedMetricDefinitionInput(data)
	if diags.HasError() {
		return nil, nil, nil, diags
	}

	queryInput := &gql.MultiStageQueryInput{
		OutputStage: ldmDefaultStageID,
		Stages:      []gql.StageQueryInput{logDef.ShapingQuery},
	}

	overwriteSource := true
	input := &gql.DatasetInput{
		OverwriteSource: &overwriteSource,
	}

	defType := gql.DatasetDefinitionTypeLogderivedmetric
	input.DatasetDefinitionType = &defType

	if v, ok := data.GetOk("name"); ok {
		input.Label = v.(string)
	} else {
		return nil, nil, nil, diag.Errorf("name not set")
	}

	input.Description = stringPtr(data.Get("description").(string))

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	return input, queryInput, logDef, diags
}

func previousLDMInputOIDVersion(data *schema.ResourceData, datasetID string) *string {
	prevOIDValue, ok := data.GetOk("input")
	if !ok {
		return nil
	}

	prevOID, err := oid.NewOID(prevOIDValue.(string))
	if err != nil || prevOID.Id != datasetID {
		return nil
	}

	return prevOID.Version
}

func logDerivedMetricDatasetToResourceData(d *gql.LogDerivedMetricDataset, data *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics

	if d.LogDerivedMetricTable == nil {
		return diag.Errorf("dataset %s is not a log-derived metric dataset (logDerivedMetricTable is null); use observe_dataset for transform-based datasets", d.Id)
	}

	ld := d.LogDerivedMetricTable

	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
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
	if err := data.Set("metric_name", ld.MetricName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("metric_type", toSnake(string(ld.MetricType))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("unit", ld.Unit); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("interval", ld.Interval.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	sq := ld.ShapingQuery
	for _, in := range sq.Input {
		if in.DatasetId != nil && *in.DatasetId != "" {
			inputOID := oid.OID{Type: oid.TypeDataset, Id: *in.DatasetId}
			if version := previousLDMInputOIDVersion(data, inputOID.Id); version != nil {
				inputOID.Version = version
			}
			if err := data.Set("input", inputOID.String()); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
			break
		}
	}
	if err := data.Set("query", sq.Pipeline); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	var fn gql.LogDerivedMetricAggregationFunction
	switch cfg := ld.Aggregation.Config.(type) {
	case *gql.LogDerivedMetricDefinitionAggregationLogDerivedMetricAggregationConfigSimpleLogDerivedMetricAggregationConfig:
		fn = cfg.Function
	default:
		diags = append(diags, diag.Errorf("unsupported aggregation config type in API response")...)
		return diags
	}

	aggBlock := map[string]interface{}{
		"function": toSnake(string(fn)),
	}
	if fp := ld.Aggregation.FieldPath; fp != nil {
		aggBlock["field_path"] = []interface{}{
			map[string]interface{}{
				"column": fp.Column,
				"path":   fp.Path,
			},
		}
	}
	if err := data.Set("aggregation", []interface{}{aggBlock}); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	tagBlocks := make([]interface{}, 0, len(ld.MetricTags))
	for _, t := range ld.MetricTags {
		tagBlocks = append(tagBlocks, map[string]interface{}{
			"name":   t.Name,
			"column": t.FieldPath.Column,
			"path":   t.FieldPath.Path,
		})
	}
	if err := data.Set("metric_tag", tagBlocks); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceLogDerivedMetricDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)
	input, queryInput, logInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	wsid, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, input, queryInput, logInput, gql.DefaultDependencyHandling())
	if err != nil {
		return diag.Errorf("failed to create log-derived metric dataset: %s", err)
	}

	data.SetId(result.Id)
	return append(diags, resourceLogDerivedMetricDatasetRead(ctx, data, meta)...)
}

func resourceLogDerivedMetricDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)
	d, err := client.GetLogDerivedMetricDataset(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read log-derived metric dataset: %s", err)
	}
	return logDerivedMetricDatasetToResourceData(d, data)
}

func resourceLogDerivedMetricDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)
	input, queryInput, logInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	id := data.Id()
	input.Id = &id
	wsid, _ := oid.NewOID(data.Get("workspace").(string))

	result, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, input, queryInput, logInput, gql.DefaultDependencyHandling())
	if err != nil {
		return diag.Errorf("failed to update log-derived metric dataset: %s", err)
	}

	return logDerivedMetricDatasetToResourceData(result, data)
}

func resourceLogDerivedMetricDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)
	if err := client.DeleteDataset(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete log-derived metric dataset: %s", err)
	}
	return nil
}
