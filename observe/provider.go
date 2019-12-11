package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_KEY", nil),
				Description: "Observe API Key from https://app.observeinc.com/#account",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_URL", "https://118647111237.observe-eng.com/v1/meta"),
				Description: "Observe API URL",
			},
		},
		ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {
			config := Config{
				BaseURL: d.Get("url").(string),
				ApiKey:  d.Get("key").(string),
			}
			return config.Client()
		},
		DataSourcesMap: map[string]*schema.Resource{
			"observe_dataset": dataSourceDataset(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_workspace": resourceWorkspace(),
			"observe_dataset":   resourceDataset(),
		},
	}
}
