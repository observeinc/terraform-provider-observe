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
	flagCacheClient       = "cache-client"
	tfSourceFormatDefault = "terraform/%s"
)

// Provider returns observe terraform provider
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"customer": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_CUSTOMER", nil),
				Description: "Observe Customer ID.",
			},
			"api_token": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("OBSERVE_API_TOKEN", nil),
				Description:   "Observe API Token. Used for authenticating requests to API in the absence of `user_email` and `user_password`.",
				ConflictsWith: []string{"user_email", "user_password"},
				Sensitive:     true,
			},
			"user_email": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("OBSERVE_USER_EMAIL", nil),
				RequiredWith: []string{"user_password"},
				Description:  "User email. Requires additionally providing `user_password`.",
			},
			"user_password": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("OBSERVE_USER_PASSWORD", nil),
				Description:  "Password for provided `user_email`.",
				RequiredWith: []string{"user_email"},
				Sensitive:    true,
			},
			"domain": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_DOMAIN", "observeinc.com"),
				Description: "Observe API domain. Defaults to `observeinc.com`.",
			},
			"insecure": {
				Type:        schema.TypeBool,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_INSECURE", false),
				Optional:    true,
				Description: "Skip TLS certificate validation.",
			},
			"retry_count": {
				Type:        schema.TypeInt,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_RETRY_COUNT", "3"),
				Optional:    true,
				Description: "Maximum number of retries on temporary network failures. Defaults to 3.",
			},
			"retry_wait": {
				Type:             schema.TypeString,
				DefaultFunc:      schema.EnvDefaultFunc("OBSERVE_RETRY_WAIT", "3s"),
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "Time between retries. Defaults to 3s.",
			},
			"flags": {
				Type:             schema.TypeString,
				DefaultFunc:      schema.EnvDefaultFunc("OBSERVE_FLAGS", ""),
				ValidateDiagFunc: validateFlags,
				Optional:         true,
				Description:      "Used to toggle experimental features.",
			},
			"http_client_timeout": {
				Type:             schema.TypeString,
				Optional:         true,
				DefaultFunc:      schema.EnvDefaultFunc("OBSERVE_HTTP_CLIENT_TIMEOUT", "2m"),
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "HTTP client timeout. Defaults to 2 minutes.",
			},
			"source_comment": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_SOURCE_COMMENT", nil),
				Description: "Source identifier comment. If null, fallback to `user_email`.",
			},
			"source_format": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_SOURCE_FORMAT", tfSourceFormatDefault),
				Description: "Source identifier format.",
			},
			"managing_object_id": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_MANAGING_OBJECT_ID", nil),
				Optional:    true,
				Description: "Managing object ID.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"observe_dataset":           dataSourceDataset(),
			"observe_link":              dataSourceLink(),
			"observe_workspace":         dataSourceWorkspace(),
			"observe_query":             dataSourceQuery(),
			"observe_board":             dataSourceBoard(),
			"observe_monitor":           dataSourceMonitor(),
			"observe_datastream":        dataSourceDatastream(),
			"observe_worksheet":         dataSourceWorksheet(),
			"observe_dashboard":         dataSourceDashboard(),
			"observe_folder":            dataSourceFolder(),
			"observe_app":               dataSourceApp(),
			"observe_default_dashboard": dataSourceDefaultDashboard(),
			"observe_terraform":         dataSourceTerraform(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_dataset":                resourceDataset(),
			"observe_source_dataset":         resourceSourceDataset(),
			"observe_link":                   resourceLink(),
			"observe_workspace":              resourceWorkspace(),
			"observe_bookmark_group":         resourceBookmarkGroup(),
			"observe_bookmark":               resourceBookmark(),
			"observe_http_post":              resourceHTTPPost(),
			"observe_channel_action":         resourceChannelAction(),
			"observe_channel":                resourceChannel(),
			"observe_monitor":                resourceMonitor(),
			"observe_board":                  resourceBoard(),
			"observe_poller":                 resourcePoller(),
			"observe_datastream":             resourceDatastream(),
			"observe_datastream_token":       resourceDatastreamToken(),
			"observe_worksheet":              resourceWorksheet(),
			"observe_dashboard":              resourceDashboard(),
			"observe_folder":                 resourceFolder(),
			"observe_app":                    resourceApp(),
			"observe_app_datasource":         resourceAppDataSource(),
			"observe_preferred_path":         resourcePreferredPath(),
			"observe_default_dashboard":      resourceDefaultDashboard(),
			"observe_layered_setting_record": resourceLayeredSettingRecord(),
		},
		TerraformVersion: version.ProviderVersion,
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

		if v, ok := data.GetOk("api_token"); ok {
			s := v.(string)
			config.ApiToken = &s
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

		tfSourceFormat := data.Get("source_format").(string)

		s := fmt.Sprintf(tfSourceFormat, "")
		if v, ok := data.GetOk("source_comment"); ok {
			s = fmt.Sprintf(tfSourceFormat, v)
		} else if config.UserEmail != nil {
			s = fmt.Sprintf(tfSourceFormat, *config.UserEmail)
		}
		config.Source = &s

		if v, ok := data.GetOk("managing_object_id"); ok {
			managingId := v.(string)
			config.ManagingObjectID = &managingId
		}

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
