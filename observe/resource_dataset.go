package observe

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
							Type:        schema.TypeString,
							Description: "",
							Optional:    true,
						},
						"dataset": {
							Type:        schema.TypeString,
							Description: "",
							Required:    true,
						},
					},
				},
				Required: true,
			},
			"pipeline": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					oldPipeline := SanitizePipeline(old)
					newPipeline := SanitizePipeline(new)

					if len(oldPipeline) != len(newPipeline) {
						return false
					}

					for i := range oldPipeline {
						if oldPipeline[i] != newPipeline[i] {
							return false
						}
					}
					return true
				},
			},
		},
	}
}

func resourceDatasetCreate(d *schema.ResourceData, c interface{}) error {
	client := c.(*Client)

	var createDatasetInput CreateDatasetInput

	if v, ok := d.GetOk("workspace"); ok {
		createDatasetInput.WorkspaceID = v.(string)
	}

	if v, ok := d.GetOk("label"); ok {
		createDatasetInput.Label = v.(string)
	}

	if v, ok := d.GetOk("pipeline"); ok {
		createDatasetInput.Pipeline = SanitizePipeline(v.(string))
	}

	if v, ok := d.GetOk("input"); ok {
		inputs := v.([]interface{})
		for n, i := range inputs {
			el := i.(map[string]interface{})

			if v, ok := el["name"]; !ok || v.(string) == "" {
				el["name"] = fmt.Sprintf("%d", n)
			}

			createDatasetInput.Inputs = append(createDatasetInput.Inputs, Input{
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
	d.Set("label", dataset.Label)
	d.Set("inputs", renameInputs(dataset.Transform.Inputs))
	d.Set("pipeline", strings.Join(dataset.Transform.Pipeline, "\n"))
	return nil
}

func resourceDatasetRead(d *schema.ResourceData, c interface{}) error {
	client := c.(*Client)

	dataset, err := client.GetDataset(d.Id())
	if err != nil {
		return err
	}

	d.Set("label", dataset.Label)
	d.Set("inputs", renameInputs(dataset.Transform.Inputs))
	d.Set("pipeline", strings.Join(dataset.Transform.Pipeline, "\n"))
	return nil
}

func renameInputs(inputs []Input) (renamed []Input) {
	for n, i := range inputs {
		el := Input{
			DatasetID: i.DatasetID,
		}
		if i.InputName != fmt.Sprintf("%d", n) {
			el.InputName = i.InputName
		} else {
			log.Printf("dumping name")
		}
		renamed = append(renamed, el)
	}
	return renamed
}

func resourceDatasetUpdate(d *schema.ResourceData, m interface{}) error {
	panic("not yet implemented")
	return resourceDatasetRead(d, m)
}

func resourceDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	client.DeleteDataset(d.Id())
	return nil
}
