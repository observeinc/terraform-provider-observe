package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"customer": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_CUSTOMER", nil),
				Description: "Observe API URL",
			},
			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("OBSERVE_TOKEN", nil),
				Description:   "Observe Token",
				ConflictsWith: []string{"user_email", "user_password"},
			},
			"user_email": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("OBSERVE_USER_EMAIL", nil),
				Description:  "Observe User Email",
				RequiredWith: []string{"user_password"},
			},
			"user_password": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("OBSERVE_USER_PASSWORD", nil),
				Description:  "Observe User Password",
				RequiredWith: []string{"user_email"},
			},
			"domain": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_DOMAIN", "observeinc.com"),
				Description: "Observe root domain",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skip TLS verification",
			},
		},
		ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {

			c := Config{
				CustomerID:   d.Get("customer").(string),
				Token:        d.Get("token").(string),
				UserEmail:    d.Get("user_email").(string),
				UserPassword: d.Get("user_password").(string),
				Domain:       d.Get("domain").(string),
				Insecure:     d.Get("insecure").(bool),
			}
			return c.Client()
		},
		DataSourcesMap: map[string]*schema.Resource{
			"observe_workspace": dataSourceWorkspace(),
			"observe_dataset":   dataSourceDataset(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_workspace": resourceWorkspace(),
			"observe_dataset":   resourceDataset(),
			"observe_transform": resourceTransform(),
		},
	}
}
