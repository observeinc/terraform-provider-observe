package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceDataset() *schema.Resource {
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
							Required: true,
						},
					},
				},
				Required: true,
			},
			"pipeline": &schema.Schema{
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
	}
}

func resourceDatasetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	var createDatasetInput observe.CreateDatasetInput

	if v, ok := d.GetOk("workspace"); ok {
		createDatasetInput.WorkspaceID = v.(string)
	}

	if v, ok := d.GetOk("label"); ok {
		createDatasetInput.Label = v.(string)
	}

	if v, ok := d.GetOk("pipeline"); ok {
		createDatasetInput.Pipeline = observe.NewPipeline(v.(string))
	}

	if v, ok := d.GetOk("input"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("%d", n)
			}

			createDatasetInput.Inputs = append(createDatasetInput.Inputs, observe.Input{
				InputName: el["name"].(string),
				DatasetID: el["dataset"].(string),
			})
		}

	}

	dataset, err := client.CreateDataset(createDatasetInput)
	if err != nil {
		return err
	}

	d.SetId(dataset.ID)

	if err := d.Set("label", dataset.Label); err != nil {
		return err
	}
	if err := d.Set("pipeline", dataset.Transform.Pipeline.String()); err != nil {
		return err
	}
	return nil
}

func resourceDatasetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)

	dataset, err := client.GetDataset(d.Id())
	if err != nil {
		return err
	}

	if err := d.Set("label", dataset.Label); err != nil {
		return err
	}
	if err := d.Set("pipeline", dataset.Transform.Pipeline.String()); err != nil {
		return err
	}
	return nil
}

func resourceDatasetUpdate(d *schema.ResourceData, m interface{}) error {
	panic("not yet implemented")
	return resourceDatasetRead(d, m)
}

func resourceDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	client.DeleteDataset(d.Id())
	return nil
}
