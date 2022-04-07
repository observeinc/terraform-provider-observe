package observe

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
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
			"datastream": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(observe.TypeDatastream),
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"chunk": &schema.Schema{
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
			"pubsub": &schema.Schema{
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
			"http": &schema.Schema{
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
			"gcp_monitoring": &schema.Schema{
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
			"mongodbatlas": &schema.Schema{
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

func newPollerConfig(data *schema.ResourceData) (config *observe.PollerConfig, diags diag.Diagnostics) {
	//TODO: handle disabling/enabling pollers
	config = &observe.PollerConfig{}

	if v, ok := data.GetOk("name"); ok {
		name := v.(string)
		config.Name = name
	}
	if v, ok := data.GetOk("retries"); ok {
		retries := int64(v.(int))
		config.Retries = &retries
	}
	if v, ok := data.GetOk("datastream"); ok {
		datastreamOID, _ := observe.NewOID(v.(string))
		config.DatastreamID = datastreamOID.ID
	}
	if v, ok := data.GetOk("interval"); ok {
		str := v.(string)
		if interval, err := time.ParseDuration(str); err != nil {
			return nil, diag.Errorf("error parsing interval: %v", err)
		} else {
			config.Interval = &interval
		}
	}
	if v, ok := data.GetOk("tags"); ok {
		config.Tags = makeStringMap(v.(map[string]interface{}))
	}
	if data.Get("chunk.#") == 1 {
		chunk := &observe.PollerChunkConfig{
			Enabled: data.Get("chunk.0.enabled").(bool),
		}
		if v, ok := data.GetOk("chunk.0.size"); ok {
			size := int64(v.(int))
			if size > 0 {
				chunk.Size = &size
			}
		}
		config.Chunk = chunk
	}
	if data.Get("pubsub.#") == 1 {
		config.PubsubConfig = &observe.PollerPubSubConfig{
			ProjectID:      data.Get("pubsub.0.project_id").(string),
			JSONKey:        data.Get("pubsub.0.json_key").(string),
			SubscriptionID: data.Get("pubsub.0.subscription_id").(string),
		}
	}
	if data.Get("http.#") == 1 {
		httpConf := &observe.PollerHTTPConfig{
			Endpoint:    data.Get("http.0.endpoint").(string),
			ContentType: data.Get("http.0.content_type").(string),
			Headers:     makeStringMap(data.Get("http.0.headers").(map[string]interface{})),
			Requests:    expandPollerHTTPRequests(data.Get("http.0.request").([]interface{})),
			Rules:       expandPollerHTTPRules(data.Get("http.0.rule").([]interface{})),
		}

		if v, ok := data.GetOk("http.0.method"); ok {
			s := v.(string)
			httpConf.Method = &s
		}

		if v, ok := data.GetOk("http.0.body"); ok {
			s := v.(string)
			httpConf.Body = &s
		}

		if ls := data.Get("http.0.template").([]interface{}); len(ls) > 0 {
			httpConf.Template = expandPollerHTTPRequest(ls[0].(map[string]interface{}))
		}

		config.HTTPConfig = httpConf
	}
	if data.Get("gcp_monitoring.#") == 1 {
		config.GcpConfig = &observe.PollerGCPMonitoringConfig{
			ProjectID:                 data.Get("gcp_monitoring.0.project_id").(string),
			JSONKey:                   data.Get("gcp_monitoring.0.json_key").(string),
			IncludeMetricTypePrefixes: makeStrSlice(data.Get("gcp_monitoring.0.include_metric_type_prefixes").([]interface{})),
			ExcludeMetricTypePrefixes: makeStrSlice(data.Get("gcp_monitoring.0.exclude_metric_type_prefixes").([]interface{})),
		}
		if v, ok := data.GetOk("gcp_monitoring.0.rate_limit"); ok {
			rateLimit := int64(v.(int))
			config.GcpConfig.RateLimit = &rateLimit
		}
		if v, ok := data.GetOk("gcp_monitoring.0.total_limit"); ok {
			totalLimit := int64(v.(int))
			config.GcpConfig.TotalLimit = &totalLimit
		}
	}
	if data.Get("mongodbatlas.#") == 1 {
		config.MongoDBAtlasConfig = &observe.PollerMongoDBAtlasConfig{
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

func makeInterfaceMap(in map[string]string) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func resourcePollerCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newPollerConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreatePoller(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create poller: %s", err.Error())
	}

	data.SetId(result.ID)
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

	if poller.WorkspaceID != "" {
		workspaceOID := observe.OID{
			Type: observe.TypeWorkspace,
			ID:   poller.WorkspaceID,
		}
		if err := data.Set("workspace", workspaceOID.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if err := data.Set("oid", poller.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// read poller configuration
	config := poller.Config
	if err := data.Set("name", config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if config.Retries != nil {
		if err := data.Set("retries", int(*config.Retries)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.Interval != nil {
		if err := data.Set("interval", config.Interval.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.DatastreamID != "" {
		datastreamOID := observe.OID{
			Type: observe.TypeDatastream,
			ID:   config.DatastreamID,
		}
		if err := data.Set("datastream", datastreamOID.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if tags := makeInterfaceMap(config.Tags); tags != nil {
		if err := data.Set("tags", tags); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.Chunk != nil {
		chunk := map[string]interface{}{
			"enabled": config.Chunk.Enabled,
		}
		if config.Chunk.Size != nil {
			chunk["size"] = int(*config.Chunk.Size)
		}
		if err := data.Set("chunk", []interface{}{chunk}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.PubsubConfig != nil {
		ps := map[string]interface{}{
			"project_id":      config.PubsubConfig.ProjectID,
			"json_key":        config.PubsubConfig.JSONKey,
			"subscription_id": config.PubsubConfig.SubscriptionID,
		}
		if err := data.Set("pubsub", []interface{}{ps}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.HTTPConfig != nil {
		ht := map[string]interface{}{
			"endpoint":     config.HTTPConfig.Endpoint,
			"content_type": config.HTTPConfig.ContentType,
			"method":       ptrToString(config.HTTPConfig.Method),
			"body":         ptrToString(config.HTTPConfig.Body),
			"headers":      makeInterfaceMap(config.HTTPConfig.Headers),
			"template":     flattenPollerHTTPRequest(config.HTTPConfig.Template),
			"request":      flattenPollerHTTPRequests(config.HTTPConfig.Requests),
			"rule":         flattenPollerHTTPRules(config.HTTPConfig.Rules),
		}

		if err := data.Set("http", []interface{}{ht}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.GcpConfig != nil {
		gcp := map[string]interface{}{
			"project_id":  config.GcpConfig.ProjectID,
			"json_key":    config.GcpConfig.JSONKey,
			"rate_limit":  config.GcpConfig.RateLimit,
			"total_limit": config.GcpConfig.TotalLimit,
		}
		if len(config.GcpConfig.IncludeMetricTypePrefixes) != 0 {
			gcp["include_metric_type_prefixes"] = config.GcpConfig.IncludeMetricTypePrefixes
		}
		if len(config.GcpConfig.ExcludeMetricTypePrefixes) != 0 {
			gcp["exclude_metric_type_prefixes"] = config.GcpConfig.ExcludeMetricTypePrefixes
		}
		if config.GcpConfig.RateLimit != nil {
			gcp["rate_limit"] = int(*config.GcpConfig.RateLimit)
		}
		if config.GcpConfig.TotalLimit != nil {
			gcp["total_limit"] = int(*config.GcpConfig.TotalLimit)
		}
		if err := data.Set("gcp_monitoring", []interface{}{gcp}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if config.MongoDBAtlasConfig != nil {
		cfg := map[string]interface{}{
			"public_key":  config.MongoDBAtlasConfig.PublicKey,
			"private_key": config.MongoDBAtlasConfig.PrivateKey,
		}
		if len(config.MongoDBAtlasConfig.IncludeGroups) != 0 {
			cfg["include_groups"] = config.MongoDBAtlasConfig.IncludeGroups
		}
		if len(config.MongoDBAtlasConfig.ExcludeGroups) != 0 {
			cfg["exclude_groups"] = config.MongoDBAtlasConfig.ExcludeGroups
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

func flattenPollerHTTPRequests(reqs []*observe.PollerHTTPRequest) (l []interface{}) {
	if len(reqs) == 0 {
		return []interface{}{}
	}

	for _, r := range reqs {
		l = append(l, flattenPollerHTTPRequest(r)...)
	}
	return
}

func flattenPollerHTTPRequest(req *observe.PollerHTTPRequest) []interface{} {
	if req == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"url":      ptrToString(req.URL),
		"method":   ptrToString(req.Method),
		"username": ptrToString(req.Username),
		"password": ptrToString(req.Password),
		"headers":  req.Headers,
		"params":   req.Params,
	}
	return []interface{}{m}
}

func expandPollerHTTPRequests(l []interface{}) (reqs []*observe.PollerHTTPRequest) {
	if len(l) == 0 {
		return nil
	}

	for _, v := range l {
		reqs = append(reqs, expandPollerHTTPRequest(v.(map[string]interface{})))
	}

	return
}

func expandPollerHTTPRequest(m map[string]interface{}) *observe.PollerHTTPRequest {
	if len(m) == 0 {
		return nil
	}

	req := &observe.PollerHTTPRequest{
		URL:      stringToPtr(m["url"].(string)),
		Method:   stringToPtr(m["method"].(string)),
		Username: stringToPtr(m["username"].(string)),
		Password: stringToPtr(m["password"].(string)),
		Headers:  makeStringMap(m["headers"].(map[string]interface{})),
		Params:   makeStringMap(m["params"].(map[string]interface{})),
	}
	return req
}

func flattenPollerHTTPRules(rules []*observe.PollerHTTPRule) (l []interface{}) {
	if len(rules) == 0 {
		return []interface{}{}
	}

	for _, r := range rules {
		l = append(l, flattenPollerHTTPRule(r)...)
	}
	return
}

func flattenPollerHTTPRule(rule *observe.PollerHTTPRule) []interface{} {
	if rule == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"match":   flattenPollerHTTPRequest(rule.Match),
		"follow":  ptrToString(rule.Follow),
		"decoder": flattenPollerHTTPDecoder(rule.Decoder),
	}
	return []interface{}{m}
}

func expandPollerHTTPRules(l []interface{}) (rules []*observe.PollerHTTPRule) {
	if len(l) == 0 {
		return nil
	}

	for _, v := range l {
		rules = append(rules, expandPollerHTTPRule(v.(map[string]interface{})))
	}
	return
}

func expandPollerHTTPRule(m map[string]interface{}) *observe.PollerHTTPRule {
	if len(m) == 0 {
		return nil
	}

	var match *observe.PollerHTTPRequest
	if ls := m["match"].([]interface{}); len(ls) > 0 {
		match = expandPollerHTTPRequest(ls[0].(map[string]interface{}))
	}

	var decoder *observe.PollerHTTPDecoder
	if ls := m["decoder"].([]interface{}); len(ls) > 0 {
		decoder = expandPollerHTTPDecoder(ls[0].(map[string]interface{}))
	}

	rule := &observe.PollerHTTPRule{
		Match:   match,
		Decoder: decoder,
	}

	// empty string is not a valid JMESPath expression
	if m["follow"] != "" {
		rule.Follow = stringToPtr(m["follow"].(string))
	}

	return rule
}

func flattenPollerHTTPDecoder(decoder *observe.PollerHTTPDecoder) []interface{} {
	if decoder == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"type": decoder.Type,
	}
	return []interface{}{m}
}

func expandPollerHTTPDecoder(m map[string]interface{}) *observe.PollerHTTPDecoder {
	if len(m) == 0 {
		return nil
	}

	decoder := &observe.PollerHTTPDecoder{
		Type: m["type"].(string),
	}
	return decoder
}

func ptrToString(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

func stringToPtr(v string) *string {
	return &v
}
