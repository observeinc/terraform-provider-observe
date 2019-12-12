package observe

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dustinkirkland/golang-petname"
	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
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
			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"freshness": &schema.Schema{
				Type:         schema.TypeInt,
				Description:  "Desired freshness in seconds",
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
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
							StateFunc: func(v interface{}) string {
								return observe.NewPipeline(v.(string)).String()
							},
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return observe.NewPipeline(old).String() == observe.NewPipeline(new).String()
							},
						},
					},
				},
				Required: true,
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
	if v, ok := d.GetOk("label"); ok {
		c.Label = v.(string)
	} else {
		c.Label = strings.ToLower(petname.Generate(2, "-"))
	}

	if v, ok := d.GetOk("freshness"); ok {
		value := int64(v.(int)) * 1000000000
		c.FreshnessDesired = &value
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
	if err := d.Set("label", o.Config.Label); err != nil {
		return err
	}

	if o.Config.FreshnessDesired != nil {
		secs := int(*o.Config.FreshnessDesired / 1000000000)
		if err := d.Set("freshness", secs); err != nil {
			return err
		}
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
