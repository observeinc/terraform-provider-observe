package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var validPollerKinds = []string{
	"pubsub",
	"http",
	"gcp_monitoring",
	"mongodbatlas",
}

func requestResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"method": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateDiagFunc: validateStringInSlice([]string{
					http.MethodGet,
					http.MethodPut,
					http.MethodPost,
				}, true),
			},
			"headers": {
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateDiagFunc: validateMapValues(validateIsString()),
			},
			"params": {
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateDiagFunc: validateMapValues(validateIsString()),
			},
		},
	}
}

func resourcePoller() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePollerCreate,
		ReadContext:   resourcePollerRead,
		UpdateContext: resourcePollerUpdate,
		DeleteContext: resourcePollerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kind": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"retries": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"datastream": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeDatastream),
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"chunk": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"tags": {
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateDiagFunc: validateMapValues(validateIsString()),
			},
			"pubsub": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: validPollerKinds,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"json_key": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateStringIsJSON,
							DiffSuppressFunc: diffSuppressJSON,
						},
						"subscription_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"http": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: validPollerKinds,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateDiagFunc: validateStringInSlice([]string{
								http.MethodGet,
								http.MethodPut,
								http.MethodPost,
							}, true),
							ConflictsWith: []string{"http.0.request"},
						},
						"body": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"http.0.request"},
						},
						"endpoint": {
							Type:          schema.TypeString,
							Optional:      true,
							Deprecated:    "use request instead",
							ConflictsWith: []string{"http.0.request"},
						},
						"content_type": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"http.0.request"},
						},
						"headers": {
							Type:             schema.TypeMap,
							Optional:         true,
							ConflictsWith:    []string{"http.0.request"},
							ValidateDiagFunc: validateMapValues(validateIsString()),
						},
						"template": {
							Type:          schema.TypeList,
							ConflictsWith: []string{"http.0.endpoint"},
							Optional:      true,
							MaxItems:      1,
							Elem:          requestResource(),
						},
						"request": {
							Type:         schema.TypeList,
							ExactlyOneOf: []string{"http.0.request", "http.0.endpoint"},
							Optional:     true,
							Elem:         requestResource(),
						},
						"rule": {
							Type:          schema.TypeList,
							Optional:      true,
							ConflictsWith: []string{"http.0.endpoint"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"match": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem:     requestResource(),
									},
									"follow": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"decoder": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"gcp_monitoring": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: validPollerKinds,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"json_key": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateStringIsJSON,
							DiffSuppressFunc: diffSuppressJSON,
						},
						"include_metric_type_prefixes": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"exclude_metric_type_prefixes": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"rate_limit": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"total_limit": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"mongodbatlas": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: validPollerKinds,
				RequiredWith: []string{"interval"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"public_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"private_key": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
						"include_groups": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"exclude_groups": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func newPollerConfig(data *schema.ResourceData) (input *gql.PollerInput, diags diag.Diagnostics) {
	//TODO: handle disabling/enabling pollers
	input = &gql.PollerInput{}

	if v, ok := data.GetOk("name"); ok {
		input.Name = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("retries"); ok {
		input.Retries = types.Int64Scalar(int64(v.(int))).Ptr()
	}
	if v, ok := data.GetOk("datastream"); ok {
		datastreamOID, _ := oid.NewOID(v.(string))
		input.DatastreamId = &datastreamOID.Id
	}
	if v, ok := data.GetOk("interval"); ok {
		str := v.(string)
		if interval, err := time.ParseDuration(str); err != nil {
			return nil, diag.Errorf("error parsing interval: %v", err)
		} else {
			input.Interval = types.DurationScalar(interval).Ptr()
		}
	}
	if v, ok := data.GetOk("tags"); ok {
		tags, err := json.Marshal(makeStringMap(v.(map[string]interface{})))
		if err != nil {
			return nil, diag.Errorf("error parsing tags: %v", err)
		}
		input.Tags = types.JsonObject(tags).Ptr()
	}
	if data.Get("chunk.#") == 1 {
		chunk := gql.PollerChunkInput{
			Enabled: data.Get("chunk.0.enabled").(bool),
		}
		if v, ok := data.GetOk("chunk.0.size"); ok {
			size := int64(v.(int))
			if size > 0 {
				parsedChunkSize := types.Int64Scalar(size)
				chunk.Size = &parsedChunkSize
			}
		}
		input.Chunk = &chunk
	}
	if data.Get("pubsub.#") == 1 {
		input.PubsubConfig = &gql.PollerPubSubInput{
			ProjectId:      data.Get("pubsub.0.project_id").(string),
			JsonKey:        types.JsonObject(data.Get("pubsub.0.json_key").(string)),
			SubscriptionId: data.Get("pubsub.0.subscription_id").(string),
		}
	}
	if data.Get("http.#") == 1 {
		headers, err := json.Marshal(makeStringMap(data.Get("http.0.headers").(map[string]interface{})))
		if err != nil {
			return nil, diag.Errorf("error parsing HTTP headers: %v", err)
		}
		contentType := data.Get("http.0.content_type").(string)
		parsedHeaders := types.JsonObject(headers)
		httpConf := gql.PollerHTTPInput{
			ContentType: &contentType,
			Headers:     &parsedHeaders,
			Requests:    expandPollerHTTPRequests(data.Get("http.0.request").([]interface{})),
			Rules:       expandPollerHTTPRules(data.Get("http.0.rule").([]interface{})),
		}

		if v, ok := data.GetOk("http.0.endpoint"); ok {
			httpConf.Endpoint = stringPtr(v.(string))
		}

		if v, ok := data.GetOk("http.0.method"); ok {
			httpConf.Method = stringPtr(v.(string))
		}

		if v, ok := data.GetOk("http.0.body"); ok {
			httpConf.Body = stringPtr(v.(string))
		}

		if ls := data.Get("http.0.template").([]interface{}); len(ls) > 0 {
			template := expandPollerHTTPRequest(ls[0].(map[string]interface{}))
			httpConf.Template = template
		}

		input.HttpConfig = &httpConf
	}
	if data.Get("gcp_monitoring.#") == 1 {
		input.GcpConfig = &gql.PollerGCPMonitoringInput{
			ProjectId:                 data.Get("gcp_monitoring.0.project_id").(string),
			JsonKey:                   types.JsonObject(data.Get("gcp_monitoring.0.json_key").(string)),
			IncludeMetricTypePrefixes: makeStrSlice(data.Get("gcp_monitoring.0.include_metric_type_prefixes").([]interface{})),
			ExcludeMetricTypePrefixes: makeStrSlice(data.Get("gcp_monitoring.0.exclude_metric_type_prefixes").([]interface{})),
		}
		if v, ok := data.GetOk("gcp_monitoring.0.rate_limit"); ok {
			input.GcpConfig.RateLimit = types.Int64Scalar(int64(v.(int))).Ptr()
		}
		if v, ok := data.GetOk("gcp_monitoring.0.total_limit"); ok {
			input.GcpConfig.TotalLimit = types.Int64Scalar(int64(v.(int))).Ptr()
		}
	}
	if data.Get("mongodbatlas.#") == 1 {
		input.MongoDBAtlasConfig = &gql.PollerMongoDBAtlasInput{
			PublicKey:     data.Get("mongodbatlas.0.public_key").(string),
			PrivateKey:    data.Get("mongodbatlas.0.private_key").(string),
			IncludeGroups: makeStrSlice(data.Get("mongodbatlas.0.include_groups").([]interface{})),
			ExcludeGroups: makeStrSlice(data.Get("mongodbatlas.0.exclude_groups").([]interface{})),
		}
	}

	return
}

func makeStrSlice(in []interface{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, val := range in {
		out[i] = val.(string)
	}
	return out
}

func makeStringMap(in map[string]interface{}) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, val := range in {
		out[key] = val.(string)
	}
	return out
}

func resourcePollerCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newPollerConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreatePoller(ctx, id.Id, config)
	if err != nil {
		return diag.Errorf("failed to create poller: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourcePollerRead(ctx, data, meta)...)
}

func resourcePollerUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newPollerConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdatePoller(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update poller: %s", err.Error())
	}
	return append(diags, resourcePollerRead(ctx, data, meta)...)
}

func resourcePollerRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	poller, err := client.GetPoller(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read poller: %s", err.Error())
	}

	if poller.WorkspaceId != "" {
		if err := data.Set("workspace", oid.WorkspaceOid(poller.WorkspaceId).String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if err := data.Set("oid", poller.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// read poller configuration
	config := poller.Config
	if err := data.Set("name", config.GetName()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if config.GetRetries() != nil {
		if err := data.Set("retries", int(*config.GetRetries())); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.GetInterval() != nil {
		if err := data.Set("interval", config.GetInterval().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if poller.DatastreamId != nil {
		if err := data.Set("datastream", oid.DatastreamOid(*poller.DatastreamId).String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.GetTags() != nil {
		var tags map[string]interface{}
		if err := json.Unmarshal([]byte(*config.GetTags()), &tags); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
		if err := data.Set("tags", tags); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.GetChunk() != nil {
		chunk := map[string]interface{}{
			"enabled": config.GetChunk().Enabled,
		}
		if config.GetChunk().Size != nil {
			chunk["size"] = int(*config.GetChunk().Size)
		}
		if err := data.Set("chunk", []interface{}{chunk}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if pubSubConfig, ok := config.(*gql.PollerConfigPollerPubSubConfig); ok {
		ps := map[string]interface{}{
			"project_id":      pubSubConfig.ProjectId,
			"json_key":        pubSubConfig.JsonKey,
			"subscription_id": pubSubConfig.SubscriptionId,
		}
		if err := data.Set("pubsub", []interface{}{ps}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if httpConfig, ok := config.(*gql.PollerConfigPollerHTTPConfig); ok {
		ht := map[string]interface{}{
			"endpoint":     httpConfig.Endpoint,
			"content_type": httpConfig.ContentType,
			"method":       httpConfig.Method,
			"body":         httpConfig.Body,
		}
		if httpConfig.Headers != nil {
			if headers, err := httpConfig.Headers.Map(); err != nil {
				diagErr := fmt.Errorf("couldn't parse headers response as JSON object: %w", err)
				diags = append(diags, diag.FromErr(diagErr)...)
			} else {
				ht["headers"] = headers
			}
		}

		template, templateDiags := flattenPollerHTTPRequest(httpConfig.Template)
		diags = append(diags, templateDiags...)
		if !templateDiags.HasError() && template != nil {
			ht["template"] = []interface{}{template}
		}

		request, requestDiags := flattenPollerHTTPRequests(httpConfig.Requests)
		diags = append(diags, requestDiags...)
		if !requestDiags.HasError() {
			ht["request"] = request
		}

		rule, ruleDiags := flattenPollerHTTPRules(httpConfig.Rules)
		diags = append(diags, ruleDiags...)
		if !ruleDiags.HasError() {
			ht["rule"] = rule
		}

		if err := data.Set("http", []interface{}{ht}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if gcpConfig, ok := config.(*gql.PollerConfigPollerGCPMonitoringConfig); ok {
		gcp := map[string]interface{}{
			"project_id": gcpConfig.ProjectId,
		}
		if _, err := gcpConfig.JsonKey.Map(); err != nil {
			diagErr := fmt.Errorf("couldn't parse JSON key as JSON object: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else {
			gcp["json_key"] = gcpConfig.JsonKey.String()
		}
		if len(gcpConfig.IncludeMetricTypePrefixes) != 0 {
			gcp["include_metric_type_prefixes"] = gcpConfig.IncludeMetricTypePrefixes
		}
		if len(gcpConfig.ExcludeMetricTypePrefixes) != 0 {
			gcp["exclude_metric_type_prefixes"] = gcpConfig.ExcludeMetricTypePrefixes
		}
		if gcpConfig.RateLimit != nil {
			gcp["rate_limit"] = int(*gcpConfig.RateLimit)
		}
		if gcpConfig.TotalLimit != nil {
			gcp["total_limit"] = int(*gcpConfig.TotalLimit)
		}
		if err := data.Set("gcp_monitoring", []interface{}{gcp}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if mongoDbAtlasConfig, ok := config.(*gql.PollerConfigPollerMongoDBAtlasConfig); ok {
		cfg := map[string]interface{}{
			"public_key":  mongoDbAtlasConfig.PublicKey,
			"private_key": mongoDbAtlasConfig.PrivateKey,
		}
		if len(mongoDbAtlasConfig.IncludeGroups) != 0 {
			cfg["include_groups"] = mongoDbAtlasConfig.IncludeGroups
		}
		if len(mongoDbAtlasConfig.ExcludeGroups) != 0 {
			cfg["exclude_groups"] = mongoDbAtlasConfig.ExcludeGroups
		}
		if err := data.Set("mongodbatlas", []interface{}{cfg}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	return diags
}

func resourcePollerDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeletePoller(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete poller: %s", err.Error())
	}
	return diags
}

func flattenPollerHTTPRequests(reqs []gql.HttpRequestConfig) (flats []map[string]interface{}, diags diag.Diagnostics) {
	if len(reqs) == 0 {
		return []map[string]interface{}{}, nil
	}

	for _, r := range reqs {
		flat, diag := flattenPollerHTTPRequest(&r)
		diags = append(diags, diag...)
		flats = append(flats, flat)
	}

	return
}

func flattenPollerHTTPRequest(req *gql.HttpRequestConfig) (flat map[string]interface{}, diags diag.Diagnostics) {
	if req == nil {
		return
	}

	flat = map[string]interface{}{
		"url":      req.Url,
		"method":   req.Method,
		"username": req.Username,
		"password": req.Password,
	}

	if req.Headers != nil {
		if headers, err := req.Headers.Map(); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		} else {
			flat["headers"] = headers
		}
	}
	if req.Params != nil {
		if params, err := req.Params.Map(); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		} else {
			flat["params"] = params
		}
	}

	return
}

func expandPollerHTTPRequests(l []interface{}) (reqs []gql.PollerHTTPRequestInput) {
	if len(l) == 0 {
		return nil
	}

	for _, v := range l {
		if req := expandPollerHTTPRequest(v.(map[string]interface{})); req != nil {
			reqs = append(reqs, *req)
		}
	}

	return
}

func expandPollerHTTPRequest(m map[string]interface{}) *gql.PollerHTTPRequestInput {
	if len(m) == 0 {
		return nil
	}

	headers, _ := json.Marshal(m["headers"].(map[string]interface{}))
	params, _ := json.Marshal(m["params"].(map[string]interface{}))

	url := m["url"].(string)
	method := m["method"].(string)
	username := m["username"].(string)
	password := m["password"].(string)
	parsedHeaders := types.JsonObject(headers)
	parsedParams := types.JsonObject(params)
	if len(headers) > 2 {
		func() {

		}()
	}

	req := &gql.PollerHTTPRequestInput{
		Url:      &url,
		Method:   &method,
		Username: &username,
		Password: &password,
		Headers:  &parsedHeaders,
		Params:   &parsedParams,
	}
	return req
}

func flattenPollerHTTPRules(rules []gql.PollerConfigPollerHTTPConfigRulesPollerHTTPRuleConfig) (flats []map[string]interface{}, diags diag.Diagnostics) {
	if len(rules) == 0 {
		return
	}

	for _, r := range rules {
		rule, diag := flattenPollerHTTPRule(&r)
		diags = append(diags, diag...)
		if !diag.HasError() {
			flats = append(flats, rule)
		}
	}

	return
}

func flattenPollerHTTPRule(rule *gql.PollerConfigPollerHTTPConfigRulesPollerHTTPRuleConfig) (flat map[string]interface{}, diags diag.Diagnostics) {
	if rule == nil {
		return
	}

	flat = map[string]interface{}{
		"follow":  rule.Follow,
		"decoder": flattenPollerHTTPDecoder(rule.Decoder),
	}

	match, diag := flattenPollerHTTPRequest(rule.Match)
	diags = append(diags, diag...)
	if !diag.HasError() && match != nil {
		flat["match"] = []interface{}{match}
	}
	return
}

func expandPollerHTTPRules(l []interface{}) (rules []gql.PollerHTTPRuleInput) {
	if len(l) == 0 {
		return nil
	}

	for _, v := range l {
		if req := expandPollerHTTPRule(v.(map[string]interface{})); req != nil {
			rules = append(rules, *req)
		}
	}
	return
}

func expandPollerHTTPRule(m map[string]interface{}) *gql.PollerHTTPRuleInput {
	if len(m) == 0 {
		return nil
	}

	var match *gql.PollerHTTPRequestInput
	if ls := m["match"].([]interface{}); len(ls) > 0 {
		match = expandPollerHTTPRequest(ls[0].(map[string]interface{}))
	}

	var decoder *gql.PollerHTTPDecoderInput
	if ls := m["decoder"].([]interface{}); len(ls) > 0 {
		decoder = expandPollerHTTPDecoder(ls[0].(map[string]interface{}))
	}

	rule := &gql.PollerHTTPRuleInput{
		Match:   match,
		Decoder: decoder,
	}

	// empty string is not a valid JMESPath expression
	if m["follow"] != "" {
		follow := m["follow"].(string)
		rule.Follow = &follow
	}

	return rule
}

func flattenPollerHTTPDecoder(decoder *gql.PollerConfigPollerHTTPConfigRulesPollerHTTPRuleConfigDecoderPollerHTTPDecoderConfig) []interface{} {
	if decoder == nil {
		return nil
	}

	m := map[string]interface{}{
		"type": decoder.Type,
	}
	return []interface{}{m}
}

func expandPollerHTTPDecoder(m map[string]interface{}) *gql.PollerHTTPDecoderInput {
	if len(m) == 0 {
		return nil
	}

	decoder := &gql.PollerHTTPDecoderInput{
		Type: m["type"].(string),
	}
	return decoder
}
