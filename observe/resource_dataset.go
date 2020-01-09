package observe

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/mitchellh/mapstructure"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceDataset() *schema.Resource {
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
			"name": {
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
			"field": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "string",
							ValidateFunc: validation.StringInSlice(observe.FieldTypes, true),
						},
					},
				},
				Optional: true,
			},
		},
	}
}

type datasetResourceData struct {
	*schema.ResourceData
}

func (d *datasetResourceData) GetConfig() (observe.DatasetConfig, error) {
	c := observe.DatasetConfig{
		Label: d.Get("name").(string),
	}

	if v, ok := d.GetOk("freshness"); ok {
		freshness, _ := time.ParseDuration(v.(string))
		c.FreshnessDesired = &freshness
	}

	if v, ok := d.GetOk("icon_url"); ok {
		icon := v.(string)
		c.IconURL = &icon
	}

	if v, ok := d.GetOk("field"); ok {
		if err := mapstructure.Decode(v, &c.Fields); err != nil {
			return c, fmt.Errorf("failed to decode fields: %w", err)
		}
	}

	return c, nil
}

func (d *datasetResourceData) SetState(o *observe.Dataset) (err error) {
	d.SetId(o.ID)

	err = d.Set("name", o.Config.Label)

	if freshness := o.Config.FreshnessDesired; err != nil && freshness != nil {
		err = d.Set("freshness", freshness.String())
	}

	if iconURL := o.Config.IconURL; err != nil && iconURL != nil {
		err = d.Set("icon_url", *iconURL)
	}

	// TODO: fields
	return err
}

func resourceDatasetCreate(data *schema.ResourceData, meta interface{}) error {
	var (
		client  = meta.(*observe.Client)
		dataset = &datasetResourceData{data}
	)

	config, err := dataset.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.CreateDataset(data.Get("workspace").(string), config)
	if err != nil {
		return err
	}

	return dataset.SetState(result)
}

func resourceDatasetRead(data *schema.ResourceData, meta interface{}) error {
	var (
		client  = meta.(*observe.Client)
		dataset = &datasetResourceData{data}
	)

	result, err := client.GetDataset(dataset.Id())
	if err != nil {
		return err
	}

	return dataset.SetState(result)
}

func resourceDatasetUpdate(data *schema.ResourceData, meta interface{}) error {
	var (
		client  = meta.(*observe.Client)
		dataset = &datasetResourceData{data}
	)

	config, err := dataset.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.UpdateDataset(data.Get("workspace").(string), dataset.Id(), config)
	if err != nil {
		return err
	}

	return dataset.SetState(result)
}

func resourceDatasetDelete(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	return client.DeleteDataset(data.Id())
}
