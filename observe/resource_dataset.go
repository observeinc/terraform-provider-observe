package observe

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/mitchellh/mapstructure"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func mergeSchema(kvs ...map[string]*schema.Schema) map[string]*schema.Schema {
	result := make(map[string]*schema.Schema)
	for _, schema := range kvs {
		for k, v := range schema {
			if _, ok := result[k]; ok {
				panic(fmt.Sprintf("schema defines multiple values for %s", k))
			}
			result[k] = v
		}
	}
	return result
}

func resourceDataset() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatasetCreate,
		Read:   resourceDatasetRead,
		Update: resourceDatasetUpdate,
		Delete: resourceDatasetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchema(map[string]*schema.Schema{
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
			"keys": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Optional: true,
			},
		}, getTransformSchema(true)),
	}
}

type datasetResourceData struct {
	*schema.ResourceData
}

func (d *datasetResourceData) GetConfig() (observe.DatasetConfig, error) {
	c := observe.DatasetConfig{
		Name: d.Get("name").(string),
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

func (d *datasetResourceData) SetState(o *observe.Dataset) error {
	d.SetId(o.ID)

	c := o.Config

	if err := d.Set("name", c.Name); err != nil {
		return err
	}

	if freshness := c.FreshnessDesired; freshness != nil {
		if err := d.Set("freshness", freshness.String()); err != nil {
			return err
		}
	}

	if iconURL := c.IconURL; iconURL != nil {
		if err := d.Set("icon_url", *iconURL); err != nil {
			return err
		}
	}

	/* XXX: missing, because we don't yet know what fields we pre-declared.
	var fields []interface{}
	for _, f := range c.Fields {
		field := map[string]interface{}{
			"name": f.Name,
			"type": f.Type,
		}

		fields = append(fields, field)
	}

	if err := d.Set("field", fields); err != nil {
		return err
	}
	*/

	return nil
}

func resourceDatasetCreate(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		dataset   = &datasetResourceData{data}
		transform = &transformResourceData{
			ResourceData: data,
			Embedded:     true,
		}
	)

	config, err := dataset.GetConfig()
	if err != nil {
		return err
	}

	transformConfig, err := transform.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.CreateDataset(data.Get("workspace").(string), config)
	if err != nil {
		return err
	}

	if err := dataset.SetState(result); err != nil {
		return err
	}

	if transformConfig != nil {
		transformResult, err := client.SetTransform(result.ID, transformConfig)
		if err != nil {
			return err
		}
		return transform.SetState(transformResult)
	}

	return nil
}

func resourceDatasetRead(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		dataset   = &datasetResourceData{data}
		transform = &transformResourceData{
			ResourceData: data,
			Embedded:     true,
		}
	)

	result, err := client.GetDataset(dataset.Id())
	if err != nil {
		return err
	}

	// no dataset found
	if result == nil {
		return nil
	}

	if err := dataset.SetState(result); err != nil {
		return err
	}

	transformResult, err := client.GetTransform(dataset.Id())
	if err != nil {
		return err
	}

	if transformResult != nil {
		return transform.SetState(transformResult)
	}

	return nil
}

func resourceDatasetUpdate(data *schema.ResourceData, meta interface{}) error {
	var (
		client    = meta.(*observe.Client)
		dataset   = &datasetResourceData{data}
		transform = &transformResourceData{
			ResourceData: data,
			Embedded:     true,
		}
	)

	config, err := dataset.GetConfig()
	if err != nil {
		return err
	}

	transformConfig, err := transform.GetConfig()
	if err != nil {
		return err
	}

	result, err := client.UpdateDataset(data.Get("workspace").(string), dataset.Id(), config)
	if err != nil {
		return err
	}

	if err := dataset.SetState(result); err != nil {
		return err
	}

	if transformConfig != nil || data.HasChange("stage") {
		transformResult, err := client.SetTransform(result.ID, transformConfig)
		if err != nil {
			return err
		}
		return transform.SetState(transformResult)
	}

	return nil
}

func resourceDatasetDelete(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*observe.Client)
	return client.DeleteDataset(data.Id())
}
