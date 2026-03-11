package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceLogDerivedMetricDataset() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a log derived metric dataset. Log derived metrics allow you to create metrics from log data.",
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
				Description:      "The name of the log derived metric dataset",
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
			"metric_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the metric to be created",
			},
			"metric_type": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "gauge",
				ValidateDiagFunc: validateEnums(gql.AllMetricTypes),
				DiffSuppressFunc: diffSuppressEnums,
				Description:      "The type of metric (gauge, counter, cumulativeCounter)",
			},
			"unit": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The unit of measurement for the metric",
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1m",
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "The aggregation interval for the metric",
			},
			"input_dataset": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				Description:      "The input dataset OID to derive metrics from",
			},
			"shaping_query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "OPAL query to shape/filter the input data before aggregation",
			},
			"aggregation": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Aggregation configuration for the metric",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Aggregation function (count, count_distinct, sum, avg, min, max)",
						},
						"field_column": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Column name for the field to aggregate (required for sum, avg, min, max)",
						},
						"field_path": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     ".",
							Description: "Path within the column (default: '.')",
						},
					},
				},
			},
			"metric_tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Tags/dimensions for the metric",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Tag name",
						},
						"field_column": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Column name for the tag value",
						},
						"field_path": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     ".",
							Description: "Path within the column (default: '.')",
						},
					},
				},
			},
		},
	}
}

// newLogDerivedMetricDatasetConfig builds the API inputs from Terraform schema
func newLogDerivedMetricDatasetConfig(data ResourceReader) (*gql.DatasetInput, *gql.LogDerivedMetricDefinitionInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	overwriteSource := true
	datasetInput := &gql.DatasetInput{
		OverwriteSource: &overwriteSource,
	}

	// Set dataset definition type
	datasetDefType := gql.DatasetDefinitionTypeLogderivedmetric
	datasetInput.DatasetDefinitionType = &datasetDefType

	// Basic dataset fields
	if v, ok := data.GetOk("name"); ok {
		datasetInput.Label = v.(string)
	} else {
		return nil, nil, diag.Errorf("name not set")
	}

	datasetInput.Description = stringPtr(data.Get("description").(string))

	if v, ok := data.GetOk("icon_url"); ok {
		datasetInput.IconUrl = stringPtr(v.(string))
	}

	// Build log derived metric definition
	logMetricInput := &gql.LogDerivedMetricDefinitionInput{}

	if v, ok := data.GetOk("metric_name"); ok {
		logMetricInput.MetricName = v.(string)
	} else {
		return nil, nil, diag.Errorf("metric_name not set")
	}

	if v, ok := data.GetOk("metric_type"); ok {
		metricType := gql.MetricType(toCamel(v.(string)))
		logMetricInput.MetricType = &metricType
	}

	if v, ok := data.GetOk("unit"); ok {
		unit := v.(string)
		logMetricInput.Unit = &unit
	}

	if v, ok := data.GetOk("interval"); ok {
		duration, _ := types.ParseDurationScalar(v.(string))
		logMetricInput.Interval = duration
	}

	// Build shaping query
	if v, ok := data.GetOk("shaping_query"); ok {
		pipeline := v.(string)

		// Get input dataset
		var inputDatasetId string
		if v, ok := data.GetOk("input_dataset"); ok {
			inputOid, err := oid.NewOID(v.(string))
			if err != nil {
				return nil, nil, diag.FromErr(err)
			}
			inputDatasetId = inputOid.Id
		} else {
			return nil, nil, diag.Errorf("input_dataset not set")
		}

		logMetricInput.ShapingQuery = gql.StageQueryInput{
			Pipeline: pipeline,
			Input: []gql.InputDefinitionInput{
				{
					InputName: "input",
					DatasetId: &inputDatasetId,
				},
			},
		}
	} else {
		return nil, nil, diag.Errorf("shaping_query not set")
	}

	// Build aggregation
	if v, ok := data.GetOk("aggregation"); ok {
		aggList := v.([]interface{})
		if len(aggList) > 0 {
			aggMap := aggList[0].(map[string]interface{})

			function := gql.LogDerivedMetricAggregationFunction(toCamel(aggMap["function"].(string)))
			config := gql.LogDerivedMetricAggregationConfigInput{
				Function: function,
			}

			aggregation := gql.LogDerivedMetricAggregationInput{
				Config: config,
			}

			// Add field path if provided
			if column, ok := aggMap["field_column"].(string); ok && column != "" {
				path := "."
				if p, ok := aggMap["field_path"].(string); ok && p != "" {
					path = p
				}
				aggregation.FieldPath = &gql.MetricTagPathInput{
					Column: column,
					Path:   path,
				}
			}

			logMetricInput.Aggregation = aggregation
		} else {
			return nil, nil, diag.Errorf("aggregation not set")
		}
	} else {
		return nil, nil, diag.Errorf("aggregation not set")
	}

	// Build metric tags
	metricTags := []gql.LogMetricTagInput{}
	if v, ok := data.GetOk("metric_tags"); ok {
		tagsList := v.([]interface{})
		for _, tagInterface := range tagsList {
			tagMap := tagInterface.(map[string]interface{})

			name := tagMap["name"].(string)
			column := tagMap["field_column"].(string)
			path := "."
			if p, ok := tagMap["field_path"].(string); ok && p != "" {
				path = p
			}

			metricTags = append(metricTags, gql.LogMetricTagInput{
				Name: name,
				FieldPath: gql.MetricTagPathInput{
					Column: column,
					Path:   path,
				},
			})
		}
	}
	logMetricInput.MetricTags = metricTags

	return datasetInput, logMetricInput, diags
}

// logDerivedMetricDatasetToResourceData flattens API response to Terraform state
func logDerivedMetricDatasetToResourceData(d *gql.LogDerivedMetricDataset, data *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics

	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Description != nil {
		if err := data.Set("description", *d.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", *d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", oid.DatasetOid(d.Id).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// Flatten log derived metric definition
	if d.LogDerivedMetricTable != nil {
		metric := d.LogDerivedMetricTable

		if err := data.Set("metric_name", metric.MetricName); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}

		if err := data.Set("metric_type", toSnake(string(metric.MetricType))); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}

		if err := data.Set("unit", metric.Unit); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}

		if err := data.Set("interval", metric.Interval.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}

		// Flatten shaping query
		if metric.ShapingQuery.Pipeline != "" {
			if err := data.Set("shaping_query", metric.ShapingQuery.Pipeline); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		// Get input dataset from shaping query
		if len(metric.ShapingQuery.Input) > 0 && metric.ShapingQuery.Input[0].DatasetId != nil {
			inputOid := oid.DatasetOid(*metric.ShapingQuery.Input[0].DatasetId)
			if err := data.Set("input_dataset", inputOid.String()); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		// Flatten aggregation
		aggMap := map[string]interface{}{
			"function": toSnake(string(metric.Aggregation.Config.GetFunction())),
		}
		if metric.Aggregation.FieldPath != nil {
			aggMap["field_column"] = metric.Aggregation.FieldPath.Column
			aggMap["field_path"] = metric.Aggregation.FieldPath.Path
		}
		if err := data.Set("aggregation", []interface{}{aggMap}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}

		// Flatten metric tags
		tags := make([]interface{}, len(metric.MetricTags))
		for i, tag := range metric.MetricTags {
			tags[i] = map[string]interface{}{
				"name":         tag.Name,
				"field_column": tag.FieldPath.Column,
				"field_path":   tag.FieldPath.Path,
			}
		}
		if err := data.Set("metric_tags", tags); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

// resourceLogDerivedMetricDatasetCreate creates a new log derived metric dataset
func resourceLogDerivedMetricDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	datasetInput, logMetricInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	wsid, _ := oid.NewOID(data.Get("workspace").(string))

	result, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, datasetInput, logMetricInput, gql.DefaultDependencyHandling())
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(result.Id)
	return resourceLogDerivedMetricDatasetRead(ctx, data, meta)
}

// resourceLogDerivedMetricDatasetRead reads the current state of a log derived metric dataset
func resourceLogDerivedMetricDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	dataset, err := client.GetLogDerivedMetricDataset(ctx, data.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	return logDerivedMetricDatasetToResourceData(dataset, data)
}

// resourceLogDerivedMetricDatasetUpdate updates an existing log derived metric dataset
func resourceLogDerivedMetricDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	datasetInput, logMetricInput, diags := newLogDerivedMetricDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	// Set the ID for update
	id := data.Id()
	datasetInput.Id = &id

	wsid, _ := oid.NewOID(data.Get("workspace").(string))

	dependencyHandling := gql.DefaultDependencyHandling()

	_, err := client.SaveLogDerivedMetricDataset(ctx, wsid.Id, datasetInput, logMetricInput, dependencyHandling)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceLogDerivedMetricDatasetRead(ctx, data, meta)
}

// resourceLogDerivedMetricDatasetDelete deletes a log derived metric dataset
func resourceLogDerivedMetricDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	id, _ := oid.NewOID(data.Id())
	if err := client.DeleteDataset(ctx, id.Id); err != nil {
		return diag.FromErr(err)
	}

	data.SetId("")
	return nil
}

// resourceLogDerivedMetricDatasetCustomizeDiff performs plan-time validation
func resourceLogDerivedMetricDatasetCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	client := meta.(*observe.Client)

	// Skip dry-run validation if configured to do so
	if client.SkipDatasetDryRuns {
		return nil
	}

	// Only do server-side validation if key fields change
	if !(d.HasChange("metric_name") || d.HasChange("shaping_query") || d.HasChange("aggregation") ||
		d.HasChange("metric_tags") || d.HasChange("input_dataset") || d.HasChange("name")) {
		return nil
	}

	// Skip validation if config is not fully known (e.g., referencing resources being created)
	if !d.GetRawConfig().IsWhollyKnown() {
		return nil
	}

	wsid, _ := oid.NewOID(d.Get("workspace").(string))
	datasetInput, logMetricInput, diags := newLogDerivedMetricDatasetConfig(d)
	if diags.HasError() {
		return fmt.Errorf("invalid log derived metric dataset config: %s", concatenateDiagnosticsToStr(diags))
	}

	if id := d.Id(); id != "" {
		datasetInput.Id = &id
	}

	err := client.SaveLogDerivedMetricDatasetDryRun(ctx, wsid.Id, datasetInput, logMetricInput)
	if err != nil {
		return fmt.Errorf("log derived metric dataset save dry-run failed: %s", err.Error())
	}

	return nil
}
