package observe

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/version"
)

var (
	_ provider.Provider              = &observeFrameworkProvider{}
	_ provider.ProviderWithValidateConfig = &observeFrameworkProvider{}
)

type observeFrameworkProvider struct {
	version string
}

type observeProviderModel struct {
	Customer                   types.String `tfsdk:"customer"`
	ApiToken                   types.String `tfsdk:"api_token"`
	UserEmail                  types.String `tfsdk:"user_email"`
	UserPassword               types.String `tfsdk:"user_password"`
	Domain                     types.String `tfsdk:"domain"`
	Insecure                   types.Bool   `tfsdk:"insecure"`
	RetryCount                 types.Int64  `tfsdk:"retry_count"`
	RetryWait                  types.String `tfsdk:"retry_wait"`
	Flags                      types.String `tfsdk:"flags"`
	HTTPClientTimeout          types.String `tfsdk:"http_client_timeout"`
	SourceComment              types.String `tfsdk:"source_comment"`
	SourceFormat               types.String `tfsdk:"source_format"`
	ManagingObjectID           types.String `tfsdk:"managing_object_id"`
	ExportObjectBindings       types.Bool   `tfsdk:"export_object_bindings"`
	DefaultRematerializationMode types.String `tfsdk:"default_rematerialization_mode"`
	SkipDatasetDryRuns         types.Bool   `tfsdk:"skip_dataset_dry_runs"`
}

func NewFrameworkProvider() provider.Provider {
	return &observeFrameworkProvider{
		version: version.ProviderVersion,
	}
}

func (p *observeFrameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "observe"
	resp.Version = p.version
}

func (p *observeFrameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"customer": schema.StringAttribute{
				Required:    true,
				Description: "Your Observe Customer ID.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "An Observe API Token. Used for authenticating requests to API in the absence of `user_email` and `user_password`.",
			},
			"user_email": schema.StringAttribute{
				Optional:    true,
				Description: "User email. If supplied, `user_password` is also required.",
			},
			"user_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for provided `user_email`.",
			},
			"domain": schema.StringAttribute{
				Optional:    true,
				Description: "Observe API domain. Defaults to `observeinc.com`.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate validation.",
			},
			"retry_count": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of retries on temporary network failures. Defaults to 3.",
			},
		"retry_wait": schema.StringAttribute{
			Optional:    true,
			Description: "Time between retries. Defaults to 3s.",
			Validators:  []validator.String{validateFWTimeDuration()},
		},
		"flags": schema.StringAttribute{
			Optional:    true,
			Description: "Toggle experimental features.",
			Validators:  []validator.String{validateFWFlags()},
		},
		"http_client_timeout": schema.StringAttribute{
			Optional:    true,
			Description: "HTTP client timeout. Defaults to 2m.",
			Validators:  []validator.String{validateFWTimeDuration()},
		},
			"source_comment": schema.StringAttribute{
				Optional:    true,
				Description: "Source identifier comment. If null, fallback to `user_email`.",
			},
			"source_format": schema.StringAttribute{
				Optional:    true,
				Description: "Source identifier format.",
			},
			"managing_object_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of an Observe object that serves as the parent (managing) object for all resources created by the provider (internal use).",
			},
			"export_object_bindings": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable generating object ID-name bindings for cross-tenant export/import (internal use).",
			},
		"default_rematerialization_mode": schema.StringAttribute{
			Optional:    true,
			Description: "Default rematerialization mode for datasets (internal use).",
			Validators:  []validator.String{validateFWEnums(AllRematerializationModes)},
		},
			"skip_dataset_dry_runs": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip making dry run API requests for dataset changes during the plan stage (for validation). This can speed up plan time, but means that certain classes of errors will not be detected until applying the changes (such as invalid OPAL).",
			},
		},
	}
}

func (p *observeFrameworkProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data observeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasToken := !data.ApiToken.IsNull() && !data.ApiToken.IsUnknown()
	hasEmail := !data.UserEmail.IsNull() && !data.UserEmail.IsUnknown()
	hasPassword := !data.UserPassword.IsNull() && !data.UserPassword.IsUnknown()

	if hasToken && (hasEmail || hasPassword) {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Conflicting authentication",
			"\"api_token\" conflicts with \"user_email\" and \"user_password\". Provide either an API token or user credentials, not both.",
		)
	}

	if hasEmail && !hasPassword {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_password"),
			"Missing required attribute",
			"\"user_password\" is required when \"user_email\" is specified.",
		)
	}
	if hasPassword && !hasEmail {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_email"),
			"Missing required attribute",
			"\"user_email\" is required when \"user_password\" is specified.",
		)
	}
}

// cachedFrameworkClients mirrors the caching behavior of the SDKv2 provider.
var cachedFrameworkClients sync.Map

func (p *observeFrameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data observeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customer := envOrDefault(data.Customer, "OBSERVE_CUSTOMER", "")
	domain := envOrDefault(data.Domain, "OBSERVE_DOMAIN", "observeinc.com")
	retryCountStr := envOrDefault(types.StringNull(), "OBSERVE_RETRY_COUNT", "3")
	retryCount, _ := strconv.Atoi(retryCountStr)
	if !data.RetryCount.IsNull() && !data.RetryCount.IsUnknown() {
		retryCount = int(data.RetryCount.ValueInt64())
	}

	ua := fmt.Sprintf("terraform-provider-observe/%s", p.version)
	config := &observe.Config{
		CustomerID: customer,
		Domain:     domain,
		UserAgent:  &ua,
		RetryCount: retryCount,
	}

	if s := stringValueOrEnv(data.ApiToken, "OBSERVE_API_TOKEN"); s != "" {
		config.ApiToken = &s
	}
	if s := stringValueOrEnv(data.UserEmail, "OBSERVE_USER_EMAIL"); s != "" {
		config.UserEmail = &s
	}
	if s := stringValueOrEnv(data.UserPassword, "OBSERVE_USER_PASSWORD"); s != "" {
		config.UserPassword = &s
	}

	insecureStr := os.Getenv("OBSERVE_INSECURE")
	if !data.Insecure.IsNull() && !data.Insecure.IsUnknown() {
		config.Insecure = data.Insecure.ValueBool()
	} else if insecureStr != "" {
		config.Insecure, _ = strconv.ParseBool(insecureStr)
	}

	retryWait := envOrDefault(data.RetryWait, "OBSERVE_RETRY_WAIT", "3s")
	config.RetryWait, _ = time.ParseDuration(retryWait)

	httpTimeout := envOrDefault(data.HTTPClientTimeout, "OBSERVE_HTTP_CLIENT_TIMEOUT", "2m")
	config.HTTPClientTimeout, _ = time.ParseDuration(httpTimeout)

	flagsStr := envOrDefault(data.Flags, "OBSERVE_FLAGS", "")
	config.Flags, _ = convertFlags(flagsStr)

	if config.Insecure {
		resp.Diagnostics.AddWarning("Insecure API session", "TLS certificate validation is disabled.")
	}

	sourceFormat := envOrDefault(data.SourceFormat, "OBSERVE_SOURCE_FORMAT", tfSourceFormatDefault)
	s := fmt.Sprintf(sourceFormat, "")
	if sc := stringValueOrEnv(data.SourceComment, "OBSERVE_SOURCE_COMMENT"); sc != "" {
		s = fmt.Sprintf(sourceFormat, sc)
	} else if config.UserEmail != nil {
		s = fmt.Sprintf(sourceFormat, *config.UserEmail)
	}
	config.Source = &s

	if v := stringValueOrEnv(data.ManagingObjectID, "OBSERVE_MANAGING_OBJECT_ID"); v != "" {
		config.ManagingObjectID = &v
	}

	exportStr := os.Getenv("OBSERVE_EXPORT_OBJECT_BINDINGS")
	if !data.ExportObjectBindings.IsNull() && !data.ExportObjectBindings.IsUnknown() {
		config.ExportObjectBindings = data.ExportObjectBindings.ValueBool()
	} else if exportStr != "" {
		config.ExportObjectBindings, _ = strconv.ParseBool(exportStr)
	}

	if traceparent := os.Getenv("TRACEPARENT"); traceparent != "" {
		config.TraceParent = &traceparent
	}

	if v := stringValueOrEnv(data.DefaultRematerializationMode, "OBSERVE_DEFAULT_REMATERIALIZATION_MODE"); v != "" {
		config.DefaultRematerializationMode = &v
	}

	skipDryRunsStr := os.Getenv("OBSERVE_SKIP_DATASET_DRY_RUNS")
	if !data.SkipDatasetDryRuns.IsNull() && !data.SkipDatasetDryRuns.IsUnknown() {
		config.SkipDatasetDryRuns = data.SkipDatasetDryRuns.ValueBool()
	} else if skipDryRunsStr != "" {
		config.SkipDatasetDryRuns, _ = strconv.ParseBool(skipDryRunsStr)
	}

	useCache := true
	if v, ok := config.Flags[flagCacheClient]; ok {
		useCache = v
	}

	if useCache {
		id := config.Hash()
		if cached, ok := cachedFrameworkClients.Load(id); ok {
			client := cached.(*observe.Client)
			resp.ResourceData = client
			resp.DataSourceData = client
			return
		}
	}

	client, err := observe.New(config)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create client", err.Error())
		return
	}

	if useCache {
		cachedFrameworkClients.Store(config.Hash(), client)
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *observeFrameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBookmarkResource,
	}
}

func (p *observeFrameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func envOrDefault(val types.String, envVar, defaultVal string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultVal
}

func stringValueOrEnv(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}
