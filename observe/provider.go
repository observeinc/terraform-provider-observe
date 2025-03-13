package observe

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/version"
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
				Type: schema.TypeString,

				// Since this field is required but uses an EnvDefaultFunc,
				// documentation should be generated with `env -u OBSERVE_CUSTOMER`
				// to ensure the default is unset.
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_CUSTOMER", nil),

				Description: "Your Observe Customer ID.",
			},
			"api_token": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("OBSERVE_API_TOKEN", nil),
				Description:   "An Observe API Token. Used for authenticating requests to API in the absence of `user_email` and `user_password`.",
				ConflictsWith: []string{"user_email", "user_password"},
				Sensitive:     true,
			},
			"user_email": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("OBSERVE_USER_EMAIL", nil),
				RequiredWith: []string{"user_password"},
				Description:  "User email. If supplied, `user_password` is also required.",
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
				Description:      "Toggle experimental features.",
			},
			"http_client_timeout": {
				Type:             schema.TypeString,
				Optional:         true,
				DefaultFunc:      schema.EnvDefaultFunc("OBSERVE_HTTP_CLIENT_TIMEOUT", "2m"),
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      "HTTP client timeout. Defaults to 2m.",
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
				Description: "ID of an Observe object that serves as the parent (managing) object for all resources created by the provider (internal use).",
			},
			"export_object_bindings": {
				Type:        schema.TypeBool,
				DefaultFunc: schema.EnvDefaultFunc("OBSERVE_EXPORT_OBJECT_BINDINGS", false),
				Optional:    true,
				Description: "Enable generating object ID-name bindings for cross-tenant export/import (internal use).",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"observe_dataset":           dataSourceDataset(),
			"observe_link":              dataSourceLink(),
			"observe_workspace":         dataSourceWorkspace(),
			"observe_query":             dataSourceQuery(),
			"observe_board":             dataSourceBoard(),
			"observe_monitor":           dataSourceMonitor(),
			"observe_monitor_action":    dataSourceMonitorAction(),
			"observe_datastream":        dataSourceDatastream(),
			"observe_worksheet":         dataSourceWorksheet(),
			"observe_dashboard":         dataSourceDashboard(),
			"observe_folder":            dataSourceFolder(),
			"observe_app":               dataSourceApp(),
			"observe_app_version":       dataSourceAppVersion(),
			"observe_default_dashboard": dataSourceDefaultDashboard(),
			"observe_terraform":         dataSourceTerraform(),
			"observe_oid":               dataSourceOID(),
			"observe_rbac_group":        dataSourceRbacGroup(),
			"observe_user":              dataSourceUser(),
			"observe_ingest_info":       dataSourceIngestInfo(),
			"observe_cloud_info":        dataSourceCloudInfo(),
			"observe_monitor_v2":        dataSourceMonitorV2(),
			"observe_monitor_v2_action": dataSourceMonitorV2Action(),
			"observe_reference_table":   dataSourceReferenceTable(),
			"observe_report":            dataSourceReport(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"observe_dataset":                   resourceDataset(),
			"observe_source_dataset":            resourceSourceDataset(),
			"observe_link":                      resourceLink(),
			"observe_workspace":                 resourceWorkspace(),
			"observe_bookmark_group":            resourceBookmarkGroup(),
			"observe_bookmark":                  resourceBookmark(),
			"observe_http_post":                 resourceHTTPPost(),
			"observe_channel_action":            resourceChannelAction(),
			"observe_channel":                   resourceChannel(),
			"observe_monitor_action":            resourceMonitorAction(),
			"observe_monitor_action_attachment": resourceMonitorActionAttachment(),
			"observe_monitor":                   resourceMonitor(),
			"observe_monitor_v2":                resourceMonitorV2(),
			"observe_monitor_v2_action":         resourceMonitorV2Action(),
			"observe_board":                     resourceBoard(),
			"observe_poller":                    resourcePoller(),
			"observe_datastream":                resourceDatastream(),
			"observe_datastream_token":          resourceDatastreamToken(),
			"observe_worksheet":                 resourceWorksheet(),
			"observe_dashboard":                 resourceDashboard(),
			"observe_folder":                    resourceFolder(),
			"observe_app":                       resourceApp(),
			"observe_app_datasource":            resourceAppDataSource(),
			"observe_preferred_path":            resourcePreferredPath(),
			"observe_default_dashboard":         resourceDefaultDashboard(),
			"observe_layered_setting_record":    resourceLayeredSettingRecord(),
			"observe_correlation_tag":           resourceCorrelationTag(),
			"observe_dashboard_link":            resourceDashboardLink(),
			"observe_rbac_group":                resourceRbacGroup(),
			"observe_rbac_default_group":        resourceRbacDefaultGroup(),
			"observe_rbac_group_member":         resourceRbacGroupmember(),
			"observe_rbac_statement":            resourceRbacStatement(),
			"observe_grant":                     resourceGrant(),
			"observe_resource_grants":           resourceResourceGrants(),
			"observe_filedrop":                  resourceFiledrop(),
			"observe_snowflake_outbound_share":  resourceSnowflakeOutboundShare(),
			"observe_dataset_outbound_share":    resourceDatasetOutboundShare(),
			"observe_reference_table":           resourceReferenceTable(),
			"observe_report":                    resourceReport(),
		},
		TerraformVersion: version.ProviderVersion,
	}

	// this is a bit circular: we need a client for the provider, but we need
	// userAgent to create the client, and we need the provider to get
	// userAgent. So we create provider, grab userAgent, and finally attach the
	// ConfigureContextFunc.
	userAgent := func() string {
		return provider.UserAgent("terraform-provider-observe", version.ProviderVersion)
	}

	provider.ConfigureContextFunc = getConfigureContextFunc(userAgent)
	return provider
}

func getConfigureContextFunc(userAgent func() string) schema.ConfigureContextFunc {

	// configure call is often called multiple times for same provider config,
	// causing poor reuse of underlying HTTP client. If provider config doesn't
	// change, we can reuse client.
	var cachedClients sync.Map

	return func(ctx context.Context, data *schema.ResourceData) (client interface{}, diags diag.Diagnostics) {
		ua := userAgent()
		config := &observe.Config{
			CustomerID: data.Get("customer").(string),
			Domain:     data.Get("domain").(string),
			UserAgent:  &ua,
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

		if v, ok := data.GetOk("export_object_bindings"); ok {
			config.ExportObjectBindings = v.(bool)
		}

		// trace identifier to attach to all HTTP requests in the traceparent header
		// refer https://www.w3.org/TR/trace-context/#traceparent-header
		if traceparent := os.Getenv("TRACEPARENT"); traceparent != "" {
			config.TraceParent = &traceparent
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
