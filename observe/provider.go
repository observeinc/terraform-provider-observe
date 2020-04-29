package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns observe terraform provider
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
		ConfigureContextFunc: func(ctx context.Context, data *schema.ResourceData) (client interface{}, diags diag.Diagnostics) {
			c := Config{
				CustomerID:   data.Get("customer").(string),
				Token:        data.Get("token").(string),
				UserEmail:    data.Get("user_email").(string),
				UserPassword: data.Get("user_password").(string),
				Domain:       data.Get("domain").(string),
				Insecure:     data.Get("insecure").(bool),
			}

			if c.Insecure {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Insecure API session",
				})
			}

			client, err := c.Client()
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Failed to create client",
					Detail:   err.Error(),
				})
				return nil, diags
			}
			return client, diags
		},
		DataSourcesMap: map[string]*schema.Resource{
			"observe_workspace": dataSourceWorkspace(),
			"observe_dataset":   dataSourceDataset(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_dataset":   resourceDataset(),
			"observe_fk":        resourceForeignKey(),
			"observe_workspace": resourceWorkspace(),
		},
	}
}
