package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/mitchellh/mapstructure"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceTransform() *schema.Resource {
	return &schema.Resource{
		Create: resourceTransformCreate,
		Read:   resourceTransformRead,
		Update: resourceTransformUpdate,
		Delete: resourceTransformDelete,

		Schema: map[string]*schema.Schema{
			"dataset": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"inputs": &schema.Schema{
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"references": &schema.Schema{
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"stage": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"input": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"pipeline": {
							Type:     schema.TypeString,
							Optional: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return observe.NewPipeline(old).String() == observe.NewPipeline(new).String()
							},
							StateFunc: func(val interface{}) string {
								return observe.NewPipeline(val.(string)).String()
							},
						},
					},
				},
				Required: true,
			},
		},
	}
}

type transformResourceData struct {
	*schema.ResourceData
}

func (d *transformResourceData) GetConfig() (*observe.TransformConfig, error) {
	inputs := make(map[string]string)
	references := make(map[string]string)
	var stages []*observe.Stage

	if v, ok := d.GetOk("inputs"); ok {
		for name, id := range v.(map[string]interface{}) {
			inputs[name] = id.(string)
		}
	}

	if v, ok := d.GetOk("references"); ok {
		for name, id := range v.(map[string]interface{}) {
			references[name] = id.(string)
		}
	}

	if v, ok := d.GetOk("stage"); !ok {
		return nil, fmt.Errorf("no stages found")
	} else if err := mapstructure.Decode(v, &stages); err != nil {
		return nil, fmt.Errorf("failed to decode stages: %w", err)
	}

	return observe.NewTransformConfig(inputs, references, stages...)
}

func (d *transformResourceData) SetState(o *observe.Transform) error {
	if err := d.Set("dataset", o.ID); err != nil {
		return fmt.Errorf("failed to set dataset: %w", err)
	}

	var stages []interface{}
	for _, s := range o.Stages {
		m := map[string]interface{}{"pipeline": s.Pipeline}
		if s.Name != "" {
			m["name"] = s.Name
		}
		if s.Input != "" {
			m["input"] = s.Input
		}
		stages = append(stages, m)
	}

	return d.Set("stage", stages)
}

func resourceTransformCreate(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		transform = &transformResourceData{data}
		datasetID = data.Get("dataset").(string)
	)

	config, err := transform.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.SetTransform(datasetID, config)
	if err != nil {
		return err
	}

	transform.SetId(datasetID)
	return transform.SetState(result)
}

func resourceTransformRead(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		transform = &transformResourceData{data}
	)

	result, err := client.GetTransform(transform.Id())
	if err != nil {
		return err
	}

	return transform.SetState(result)
}

func resourceTransformUpdate(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		transform = &transformResourceData{data}
		datasetID = data.Get("dataset").(string)
	)

	config, err := transform.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.SetTransform(datasetID, config)
	if err != nil {
		return err
	}

	return transform.SetState(result)
}

func resourceTransformDelete(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	_, err := client.SetTransform(data.Id(), nil)
	return err
}
