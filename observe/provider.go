package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"customer": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_CUSTOMER", nil),
				Description: "Observe API URL",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_TOKEN", nil),
				Description: "Observe Token",
			},
			"domain": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_DOMAIN", "observeinc.com"),
				Description: "Observe root domain",
			},
		},
		ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {
			config := Config{
				CustomerID: d.Get("customer").(string),
				Token:      d.Get("token").(string),
				Domain:     d.Get("domain").(string),
			}
			return config.Client()
		},
		DataSourcesMap: map[string]*schema.Resource{
			"observe_dataset": dataSourceDataset(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_workspace": resourceWorkspace(),
			"observe_transform": resourceDataset(),
		},
	}
}
