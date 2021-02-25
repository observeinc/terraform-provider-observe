package observe

import (
	"context"
	"fmt"
	"sync"
	"time"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/version"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	flagCacheClient = "cache-client"
	tfSourceFormat  = "terraform/%s"
)

// Provider returns observe terraform provider
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"customer": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_CUSTOMER", nil),
				Description: "Observe Customer ID",
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
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_INSECURE", false),
				Optional:    true,
				Description: "Skip TLS verification",
			},
			"proxy": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_PROXY", nil),
				Optional:    true,
				Description: "URL to proxy requests through",
			},
			"retry_count": {
				Type:        schema.TypeInt,
				Default:     3,
				Optional:    true,
				Description: "Maximum number of retries on temporary network failures",
			},
			"retry_wait": {
				Type:             schema.TypeString,
				Default:          "3s",
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "Time between retries",
			},
			"flags": {
				Type:             schema.TypeString,
				DefaultFunc:      schema.EnvDefaultFunc("OBSERVE_FLAGS", ""),
				ValidateDiagFunc: validateFlags,
				Optional:         true,
				Description:      "Toggle feature flags",
			},
			"http_client_timeout": {
				Type:             schema.TypeString,
				Default:          "5m",
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "HTTP client timeout",
			},
			"source_comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Source identifier comment. If null, fallback to user_email",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"observe_dataset":   dataSourceDataset(),
			"observe_fk":        dataSourceForeignKey(),
			"observe_link":      dataSourceForeignKey(),
			"observe_workspace": dataSourceWorkspace(),
			"observe_query":     dataSourceQuery(),
			"observe_board":     dataSourceBoard(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_dataset":        resourceDataset(),
			"observe_source_dataset": resourceSourceDataset(),
			"observe_fk":             resourceForeignKey(),
			"observe_link":           resourceForeignKey(),
			"observe_workspace":      resourceWorkspace(),
			"observe_bookmark_group": resourceBookmarkGroup(),
			"observe_bookmark":       resourceBookmark(),
			"observe_http_post":      resourceHTTPPost(),
			"observe_channel_action": resourceChannelAction(),
			"observe_channel":        resourceChannel(),
			"observe_monitor":        resourceMonitor(),
			"observe_board":          resourceBoard(),
		},
	}

	// this is a bit circular: we need a client for the provider, but we need
	// userAgent to create the client, and we need the provider to get
	// userAgent. So we create provider, grab userAgent, and finally attach the
	// ConfigureContextFunc.
	userAgent := provider.UserAgent("terraform-provider-observe", version.ProviderVersion)

	provider.ConfigureContextFunc = getConfigureContextFunc(userAgent)
	return provider
}

func getConfigureContextFunc(userAgent string) schema.ConfigureContextFunc {

	// configure call is often called multiple times for same provider config,
	// causing poor reuse of underlying HTTP client. If provider config doesn't
	// change, we can reuse client.
	var cachedClients sync.Map

	return func(ctx context.Context, data *schema.ResourceData) (client interface{}, diags diag.Diagnostics) {
		config := &observe.Config{
			CustomerID: data.Get("customer").(string),
			Domain:     data.Get("domain").(string),
			UserAgent:  &userAgent,
			RetryCount: data.Get("retry_count").(int),
		}

		if v, ok := data.GetOk("token"); ok {
			s := v.(string)
			config.Token = &s
		}

		if v, ok := data.GetOk("user_email"); ok {
			s := v.(string)
			config.UserEmail = &s
		}

		if v, ok := data.GetOk("user_password"); ok {
			s := v.(string)
			config.UserPassword = &s
		}

		if v, ok := data.GetOk("insecure"); ok {
			config.Insecure = v.(bool)
		}

		if v, ok := data.GetOk("proxy"); ok {
			s := v.(string)
			config.Proxy = &s
		}

		if v, ok := data.GetOk("retry_wait"); ok {
			config.RetryWait, _ = time.ParseDuration(v.(string))
		}

		if v, ok := data.GetOk("http_client_timeout"); ok {
			config.HTTPClientTimeout, _ = time.ParseDuration(v.(string))
		}

		config.Flags, _ = convertFlags(data.Get("flags").(string))

		if config.Insecure {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Insecure API session",
			})
		}

		s := fmt.Sprintf(tfSourceFormat, "")
		if v, ok := data.GetOk("source_comment"); ok {
			s = fmt.Sprintf(tfSourceFormat, v)
		} else if config.UserEmail != nil {
			s = fmt.Sprintf(tfSourceFormat, *config.UserEmail)
		}
		config.Source = &s

		// by omission, cache client
		useCache := true
		if v, ok := config.Flags[flagCacheClient]; ok {
			useCache = v
		}

		if useCache {
			id := config.Hash()
			if client, ok := cachedClients.Load(id); ok {
				return client.(*observe.Client), diags
			}

			// cache whatever client we end up returning
			defer func() {
				if !diags.HasError() {
					cachedClients.Store(id, client)
				}
			}()
		}

		client, err := observe.New(config)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to create client",
				Detail:   err.Error(),
			})
			return nil, diags
		}
		return client, diags
	}
}
