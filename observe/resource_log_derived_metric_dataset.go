package observe

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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

var allLogDerivedMetricAggregationFunctions = []gql.LogDerivedMetricAggregationFunction{
	gql.LogDerivedMetricAggregationFunctionCount,
	gql.LogDerivedMetricAggregationFunctionCountdistinct,
	gql.LogDerivedMetricAggregationFunctionSum,
	gql.LogDerivedMetricAggregationFunctionAvg,
	gql.LogDerivedMetricAggregationFunctionMin,
	gql.LogDerivedMetricAggregationFunctionMax,
}

var allMetricTypes = []gql.MetricType{
	gql.MetricTypeCumulativecounter,
	gql.MetricTypeCounter,
	gql.MetricTypeRatepersec,
	gql.MetricTypeDelta,
	gql.MetricTypeGauge,
	gql.MetricTypeTdigest,
	gql.MetricTypeSample,
	gql.MetricTypeHistogram,
	gql.MetricTypeExponentialhistogram,
}

func parseMetricTypeSnake(s string) gql.MetricType {
	for _, m := range allMetricTypes {
		if toSnake(string(m)) == s {
			return m
		}
	}
	return gql.MetricType(s)
}

func parseLogDerivedAggFnSnake(s string) gql.LogDerivedMetricAggregationFunction {
	for _, m := range allLogDerivedMetricAggregationFunctions {
		if toSnake(string(m)) == s {
			return m
		}
	}
	return gql.LogDerivedMetricAggregationFunction(toCamel(s))
}

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
				ValidateDiagFunc: validateEnums(allMetricTypes),
				DiffSuppressFunc: diffSuppressEnums,
				Description:      descriptions.Get("log_derived_metric_dataset", "schema", "metric_type"),
			},
			"unit": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "unit"),
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
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
			"shaping_query": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: descriptions.Get("log_derived_metric_dataset", "schema", "shaping_query"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"inputs": {
							Type:             schema.TypeMap,
							Required:         true,
							ValidateDiagFunc: validateMapValues(validateOID()),
							Description:      descriptions.Get("transform", "schema", "inputs"),
						},
						"pipeline": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: diffSuppressPipeline,
							Description:      descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"stage_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
					},
				},
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
							ValidateDiagFunc: validateEnums(allLogDerivedMetricAggregationFunctions),
							DiffSuppressFunc: diffSuppressEnums,
						},
						"field_path": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"column": {
										Type:     schema.TypeString,
										Required: true,
									},
									"path": {
										Type:     schema.TypeString,
										Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"column": {
							Type:     schema.TypeString,
							Required: true,
						},
						"path": {
							Type:     schema.TypeString,
							Required: true,
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

	if !(d.HasChange("shaping_query") || d.HasChange("aggregation") || d.HasChange("metric_name") ||
		d.HasChange("metric_type") || d.HasChange("unit") || d.HasChange("interval") ||
		d.HasChange("metric_tag") || d.HasChange("name")) {
		return nil
	}

	if !d.GetRawConfig().IsWhollyKnown() {
		return nil
	}

	wsid, _ := oid.NewOID(d.Get("workspace").(string))
	input, logInput, diags := newLogDerivedMetricDatasetConfig(d)
	if diags.HasError() {
		return fmt.Errorf("invalid log-derived metric dataset config: %s", concatenateDiagnosticsToStr(diags))
	}
	if id := d.Id(); id != "" {
		input.Id = &id
	}

	_, err := client.SaveLogDerivedMetricDatasetDryRun(ctx, wsid.Id, input, logInput)
	if err != nil {
		return fmt.Errorf("log-derived metric dataset save dry-run failed: %s", err.Error())
	}

	return nil
}

func newShapingStageQueryInput(data ResourceReader) (gql.StageQueryInput, diag.Diagnostics) {
	raw := data.Get("shaping_query").([]interface{})
	if len(raw) != 1 {
		return gql.StageQueryInput{}, diag.Errorf("shaping_query must have exactly one block")
	}
	m := raw[0].(map[string]interface{})

	inputMap := m["inputs"].(map[string]interface{})
	pipeline := m["pipeline"].(string)

	sortedNames := make([]string, 0, len(inputMap))
	inputs := make(map[string]*gql.InputDefinitionInput, len(inputMap))
	for name, v := range inputMap {
		is, err := oid.NewOID(v.(string))
		if err != nil {
			return gql.StageQueryInput{}, diag.FromErr(fmt.Errorf("input %q: %w", name, err))
		}
		id := is.Id
		if id == "" {
			return gql.StageQueryInput{}, diag.FromErr(fmt.Errorf("input %q: empty dataset id", name))
		}
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			return gql.StageQueryInput{}, diag.FromErr(fmt.Errorf("input %q: %w", name, errObjectIDInvalid))
		}
		inputs[name] = &gql.InputDefinitionInput{
			InputName: name,
			DatasetId: &id,
		}
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	if len(sortedNames) == 0 {
		return gql.StageQueryInput{}, diag.FromErr(errInputsMissing)
	}

	stageInput := gql.StageQueryInput{
		Pipeline: pipeline,
	}

	if v, ok := m["stage_id"]; ok && v.(string) != "" {
		s := v.(string)
		stageInput.Id = &s
	}

	// Include each declared input; also add secondary inputs referenced in pipeline via @name.
	var defaultInput *gql.InputDefinitionInput
	if len(inputs) == 1 {
		defaultInput = inputs[sortedNames[0]]
	} else {
		for _, name := range sortedNames {
			in := inputs[name]
			if strings.Contains(pipeline, fmt.Sprintf("@%s", in.InputName)) ||
				strings.Contains(pipeline, fmt.Sprintf("@%q", in.InputName)) {
				defaultInput = in
				break
			}
		}
		if defaultInput == nil {
			defaultInput = inputs[sortedNames[0]]
		}
	}

	stageInput.Input = append(stageInput.Input, *defaultInput)
	for _, name := range sortedNames {
		in := inputs[name]
		if in == defaultInput {
			continue
		}
		if strings.Contains(pipeline, fmt.Sprintf("@%s", in.InputName)) ||
			strings.Contains(pipeline, fmt.Sprintf("@%q", in.InputName)) {
			stageInput.Input = append(stageInput.Input, *in)
		}
	}

	return stageInput, nil
}

func newLogDerivedMetricDefinitionInput(data ResourceReader) (*gql.LogDerivedMetricDefinitionInput, diag.Diagnostics) {
	shaping, diags := newShapingStageQueryInput(data)
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
	fn := parseLogDerivedAggFnSnake(fnStr)

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

	var tags []gql.LogMetricTagInput
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
		mt := parseMetricTypeSnake(v.(string))
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

func newLogDerivedMetricDatasetConfig(data ResourceReader) (*gql.DatasetInput, *gql.LogDerivedMetricDefinitionInput, diag.Diagnostics) {
	logDef, diags := newLogDerivedMetricDefinitionInput(data)
	if diags.HasError() {
		return nil, nil, diags
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
		return nil, nil, diag.Errorf("name not set")
	}

	input.Description = stringPtr(data.Get("description").(string))

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	return input, logDef, diags
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
	inputs := map[string]interface{}{}
	for _, in := range sq.Input {
		if in.DatasetId != nil && *in.DatasetId != "" {
			inputs[in.InputName] = oid.OID{Type: oid.TypeDataset, Id: *in.DatasetId}.String()
		}
	}
	shapingBlock := map[string]interface{}{
		"inputs":   inputs,
		"pipeline": sq.Pipeline,
	}
	if sq.Id != nil && *sq.Id != "" {
		shapingBlock["stage_id"] = *sq.Id
	}
	if err := data.Set("shaping_query", []interface{}{shapingBlock}); err != nil {
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
	input, logInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	wsid, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, input, logInput, gql.DefaultDependencyHandling())
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
	input, logInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	id := data.Id()
	input.Id = &id
	wsid, _ := oid.NewOID(data.Get("workspace").(string))

	result, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, input, logInput, gql.DefaultDependencyHandling())
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
