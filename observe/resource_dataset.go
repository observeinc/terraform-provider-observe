package observe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dustinkirkland/golang-petname"
	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceDataset() *schema.Resource {
	petname.NonDeterministicMode()

	return &schema.Resource{
		Create: resourceDatasetCreate,
		Read:   resourceDatasetRead,
		Update: resourceDatasetUpdate,
		Delete: resourceDatasetDelete,

		Schema: map[string]*schema.Schema{
			"workspace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"stage": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"follow": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"stage.import"},
						},
						"import": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"stage.follow"},
						},
						"pipeline": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return observe.NewPipeline(old).String() == observe.NewPipeline(new).String()
							},
						},
					},
				},
				Required: true,
			},
			"dataset": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": {
							Type:     schema.TypeString,
							Required: true,
						},
						"icon_url": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"freshness": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(i interface{}, k string) ([]string, []error) {
								s := i.(string)
								if _, err := time.ParseDuration(s); err != nil {
									return nil, []error{err}
								}
								return nil, nil
							},
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								o, _ := time.ParseDuration(old)
								n, _ := time.ParseDuration(new)
								return o == n
							},
						},
					},
				},
			},
		},
	}
}

func resourceDatasetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	datasetConfig, err := getDatasetConfig(d)
	if err != nil {
		return err
	}

	transformConfig, err := getTransformConfig(d)
	if err != nil {
		return err
	}

	dataset, err := client.CreateDataset(d.Get("workspace").(string), datasetConfig, transformConfig)
	if err != nil {
		return err
	}

	d.SetId(dataset.ID)
	return datasetToResource(dataset, d)
}

func resourceDatasetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	dataset, err := client.GetDataset(d.Id())
	if err != nil {
		return err
	}

	return datasetToResource(dataset, d)
}

func getDatasetConfig(d *schema.ResourceData) (c observe.DatasetConfig, err error) {
	var m map[string]interface{}

	if v, ok := d.GetOk("dataset"); ok {
		datasets := v.([]interface{})
		if len(datasets) == 0 {
			return
		}
		m = datasets[0].(map[string]interface{})

		if v, ok := m["label"]; ok && v != "" {
			c.Label = v.(string)
		}

		if v, ok := m["freshness"]; ok && v != "" {
			freshness, _ := time.ParseDuration(v.(string))
			c.FreshnessDesired = &freshness
		}

		if v, ok := m["icon_url"]; ok {
			icon := v.(string)
			c.IconURL = &icon
		}
	} else {
		c.Label = strings.ToLower(petname.Generate(2, "-"))
	}

	return
}

func getTransformConfig(d *schema.ResourceData) (c observe.TransformConfig, err error) {
	if v, ok := d.GetOk("stage"); ok {
		stages := v.([]interface{})

		for _, i := range stages {
			var s observe.Stage
			if err = mapstructure.Decode(i, &s); err != nil {
				return
			}

			if err = c.AddStage(&s); err != nil {
				return
			}
		}
		return
	}
	return
}

func datasetToResource(o *observe.Dataset, d *schema.ResourceData) error {
	m := make(map[string]interface{})

	if label := o.Config.Label; label != "" {
		m["label"] = label
	}

	if freshness := o.Config.FreshnessDesired; freshness != nil {
		m["freshness"] = freshness.String()
	}

	if iconURL := o.Config.IconURL; iconURL != nil {
		m["icon_url"] = *iconURL
	}

	if err := d.Set("dataset", []interface{}{m}); err != nil {
		return err
	}

	var stages []interface{}
	for _, s := range o.Transform.Stages {
		var m map[string]interface{}
		if data, err := json.Marshal(s); err != nil {
			return fmt.Errorf("failed to marshal stage: %w", err)
		} else if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("failed to unmarshal stage: %w", err)
		}
		stages = append(stages, m)
	}

	if err := d.Set("stage", stages); err != nil {
		return err
	}

	return nil
}

func resourceDatasetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	datasetConfig, err := getDatasetConfig(d)
	if err != nil {
		return err
	}

	transformConfig, err := getTransformConfig(d)
	if err != nil {
		return err
	}

	dataset, err := client.UpdateDataset(d.Get("workspace").(string), d.Id(), datasetConfig, transformConfig)
	if err != nil {
		return err
	}

	return datasetToResource(dataset, d)
}

func resourceDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	return client.DeleteDataset(d.Id())
}
