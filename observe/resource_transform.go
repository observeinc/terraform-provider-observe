package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/mitchellh/mapstructure"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

// transformSchema can be embedded in dataset resource, or declared on its own
func getTransformSchema(embedded bool) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
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
						Required: true,
						//ForceNew: true,
						DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
							return observe.NewPipeline(old).String() == observe.NewPipeline(new).String()
						},
						StateFunc: func(val interface{}) string {
							return observe.NewPipeline(val.(string)).String()
						},
					},
				},
			},
			Required: !embedded,
			Optional: embedded,
		},
	}

	if !embedded {
		s["dataset"] = &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		}
	}

	return s
}

func resourceTransform() *schema.Resource {
	return &schema.Resource{
		Create: resourceTransformCreate,
		Read:   resourceTransformRead,
		Update: resourceTransformUpdate,
		Delete: resourceTransformDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: getTransformSchema(false),
	}
}

type transformResourceData struct {
	*schema.ResourceData
	Embedded bool // whether or not this schema is embedded in datasete
}

func (d *transformResourceData) GetConfig() (*observe.TransformConfig, error) {
	inputs := make(map[string]string)
	references := make(map[string]string)
	var stages []*observe.Stage

	if _, ok := d.GetOk("stage"); !ok && d.Embedded {
		// we're embedded in dataset, no stage means no transform
		return nil, nil
	}

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

	metadata := make(map[string]string)
	if d.Embedded {
		metadata["embedded"] = "true"
	}

	return observe.NewTransformConfig(inputs, references, metadata, stages...)
}

func (d *transformResourceData) SetState(o *observe.Transform) error {
	if o == nil || o.TransformConfig == nil {
		return nil
	}

	if d.Embedded {
		if s, ok := o.TransformConfig.Metadata["embedded"]; !ok || s != "true" {
			return nil
		}
	} else {
		if err := d.Set("dataset", o.ID); err != nil {
			return fmt.Errorf("failed to set dataset: %w", err)
		}
	}

	// XXX: inputs? references?

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
		transform = &transformResourceData{ResourceData: data}
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
		transform = &transformResourceData{ResourceData: data}
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
		transform = &transformResourceData{ResourceData: data}
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

	result, err := client.GetTransform(data.Id())
	if err != nil {
		return err
	}

	if result != nil && len(result.Stages) > 0 {
		_, err = client.SetTransform(data.Id(), nil)
	}
	return err
}
