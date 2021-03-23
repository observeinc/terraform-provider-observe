package observe

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceDataset() *schema.Resource {
	return &schema.Resource{
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
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"path_cost": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"freshness": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
			},
			"stage": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
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
						},
						"input": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"pipeline": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func newDatasetConfig(data *schema.ResourceData) (*observe.DatasetConfig, diag.Diagnostics) {
	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, diags
	}

	if query == nil {
		return nil, diag.Errorf("no query provided")
	}

	config := &observe.DatasetConfig{
		Name:  data.Get("name").(string),
		Query: query,
	}

	if v, ok := data.GetOk("freshness"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		config.Freshness = &t
	}

	{
		// always reset to empty string if description not set
		description := data.Get("description").(string)
		config.Description = &description
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.IconURL = &icon
	}

	if v, ok := data.GetOk("path_cost"); ok {
		config.PathCost = int64(v.(int))
	}

	if err := config.Validate(); err != nil {
		return nil, diag.FromErr(err)
	}

	return config, diags
}

func datasetToResourceData(d *observe.Dataset, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.Freshness != nil {
		if err := data.Set("freshness", d.Config.Freshness.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.Description != nil {
		if err := data.Set("description", d.Config.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.IconURL != nil {
		if err := data.Set("icon_url", d.Config.IconURL); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	var currentCost int64
	if v, ok := data.GetOk("path_cost"); ok {
		currentCost = int64(v.(int))
	}

	if d.Config.PathCost != currentCost {
		if err := data.Set("path_cost", d.Config.PathCost); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if diags.HasError() {
		return diags
	}

	if err := flattenAndSetQuery(data, d.Config.Query); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenAndSetQuery(data *schema.ResourceData, query *observe.Query) error {
	if query == nil {
		return nil
	}

	inputs := make(map[string]interface{}, len(query.Inputs))
	for name, input := range query.Inputs {
		oid := observe.OID{
			Type: observe.TypeDataset,
			ID:   *input.Dataset,
		}

		// check for existing version timestamp we can maintain
		if v, ok := data.GetOk(fmt.Sprintf("inputs.%s", name)); ok {
			prv, err := observe.NewOID(v.(string))
			if err == nil && oid.ID == prv.ID {
				oid.Version = prv.Version
			}
		}
		inputs[name] = oid.String()
	}

	if err := data.Set("inputs", inputs); err != nil {
		return err
	}

	stages := make([]interface{}, len(query.Stages))
	for i, stage := range query.Stages {
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
	config, diags := newDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateDataset(ctx, oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dataset",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
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
	config, diags := newDatasetConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateDataset(ctx, oid.ID, data.Id(), config)
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

	oldOID, err := observe.NewOID(old)
	if err != nil {
		log.Printf("[WARN] could not convert old %s %q to OID: %s\n", k, old, err)
		return false
	}

	newOID, err := observe.NewOID(new)
	if err != nil {
		log.Printf("[WARN] could not convert new %s %q to OID: %s\n", k, new, err)
		return false
	}

	// ignore version
	return oldOID.Type == newOID.Type && oldOID.ID == newOID.ID
}
