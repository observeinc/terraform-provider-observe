package observe

import (
	"fmt"
	"strings"

	"github.com/dustinkirkland/golang-petname"

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
			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"workspace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"input": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"dataset": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
					},
				},
				Required: true,
			},
			"stage": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"pipeline": {
							Type:     schema.TypeString,
							Required: true,
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

	var datasetInput observe.DatasetInput

	if v, ok := d.GetOk("workspace"); ok {
		datasetInput.WorkspaceID = v.(string)
	}

	if v, ok := d.GetOk("label"); ok {
		datasetInput.Label = v.(string)
	} else {
		datasetInput.Label = strings.ToLower(petname.Generate(2, "-"))
	}

	if v, ok := d.GetOk("input"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("i%d", n)
			}

			datasetInput.Inputs = append(datasetInput.Inputs, observe.Input{
				Name:      el["name"].(string),
				DatasetID: el["dataset"].(string),
			})
		}

	}

	if v, ok := d.GetOk("stage"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("s%d", n)
			}

			datasetInput.Stages = append(datasetInput.Stages, observe.Stage{
				Name:     el["name"].(string),
				Pipeline: observe.NewPipeline(el["pipeline"].(string)),
			})
		}
	}

	dataset, err := client.CreateDataset(datasetInput)
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

func datasetToResource(o *observe.Dataset, d *schema.ResourceData) error {
	if err := d.Set("label", o.Label); err != nil {
		return err
	}

	var inputs []interface{}
	for _, i := range o.Transform.Inputs {
		inputs = append(inputs, map[string]interface{}{
			"name":    i.Name,
			"dataset": i.DatasetID,
		})
	}

	if err := d.Set("input", inputs); err != nil {
		return err
	}

	var stages []interface{}
	for _, s := range o.Transform.Stages {
		stages = append(stages, map[string]interface{}{
			"name":     s.Name,
			"pipeline": s.Pipeline.String(),
		})
	}

	if err := d.Set("stage", stages); err != nil {
		return err
	}

	return nil
}

func resourceDatasetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	var datasetInput observe.DatasetInput

	if v, ok := d.GetOk("workspace"); ok {
		datasetInput.WorkspaceID = v.(string)
	}

	if v, ok := d.GetOk("label"); ok {
		datasetInput.Label = v.(string)
	} else {
		datasetInput.Label = strings.ToLower(petname.Generate(2, "-"))
	}

	if v, ok := d.GetOk("input"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("i%d", n)
			}

			datasetInput.Inputs = append(datasetInput.Inputs, observe.Input{
				Name:      el["name"].(string),
				DatasetID: el["dataset"].(string),
			})
		}

	}

	if v, ok := d.GetOk("stage"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("s%d", n)
			}

			datasetInput.Stages = append(datasetInput.Stages, observe.Stage{
				Name:     el["name"].(string),
				Pipeline: observe.NewPipeline(el["pipeline"].(string)),
			})
		}
	}

	dataset, err := client.UpdateDataset(d.Id(), datasetInput)
	if err != nil {
		return err
	}

	return datasetToResource(dataset, d)
}

func resourceDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	client.DeleteDataset(d.Id())
	return nil
}
