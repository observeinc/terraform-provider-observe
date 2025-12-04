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
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var pollerBlockTypes = []string{
	"pubsub",
	"http",
	"gcp_monitoring",
	"mongodbatlas",
	"cloudwatch_metrics",
	"aws_snapshot",
}

func requestResourceRegex() *schema.Resource {
	resource := requestResource()

	// method in this case is a regular expression..
	resource.Schema["method"] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}

	return resource
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
			"auth_scheme": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateEnums(gql.AllPollerHTTPRequestAuthSchemes),
				DiffSuppressFunc: diffSuppressEnums,
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
			"body": {
				Type:     schema.TypeString,
				Optional: true,
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
		Description: descriptions.Get("poller", "description"),
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Description:      descriptions.Get("common", "schema", "workspace"),
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"oid": {
				Type:        schema.TypeString,
				Description: descriptions.Get("common", "schema", "oid"),
				Computed:    true,
			},
			"kind": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: descriptions.Get("poller", "schema", "name"),
				Required:    true,
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions.Get("poller", "schema", "disabled"),
				Default:     false,
			},
			"retries": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"datastream": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      descriptions.Get("poller", "schema", "datastream"),
				ValidateDiagFunc: validateOID(oid.TypeDatastream),
			},
			"interval": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      descriptions.Get("poller", "schema", "interval"),
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"skip_external_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skips validating any provided external API credentials against their external APIs.",
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
				ExactlyOneOf: pollerBlockTypes,
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
				ExactlyOneOf: pollerBlockTypes,
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
							Deprecated:    "Use request instead to configure a list of one or more requests.",
						},
						"body": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"http.0.request"},
							Deprecated:    "Use request instead to configure a list of one or more requests.",
						},
						"endpoint": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"http.0.request"},
							Deprecated:    "Use request instead to configure a list of one or more requests.",
						},
						"content_type": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"http.0.request"},
							Deprecated:    "Use request instead to configure a list of one or more requests.",
						},
						"headers": {
							Type:             schema.TypeMap,
							Optional:         true,
							ConflictsWith:    []string{"http.0.request"},
							ValidateDiagFunc: validateMapValues(validateIsString()),
							Deprecated:       "Use request instead to configure a list of one or more requests.",
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
										Elem:     requestResourceRegex(),
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
						"timestamp": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"source": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"format": {
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: validateEnums(gql.AllPollerHTTPTimestampFormats),
										DiffSuppressFunc: diffSuppressEnums,
									},
									"offset": {
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: validateTimeDuration,
									},
									"truncate": {
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: validateTimeDuration,
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
				ExactlyOneOf: pollerBlockTypes,
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
				ExactlyOneOf: pollerBlockTypes,
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
			"cloudwatch_metrics": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: pollerBlockTypes,
				RequiredWith: []string{"interval", "datastream"},
				Description:  descriptions.Get("poller", "schema", "cloudwatch_metrics", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period": {
							Type:             schema.TypeString,
							ValidateDiagFunc: validateTimeDuration,
							DiffSuppressFunc: diffSuppressTimeDuration,
							Optional:         true,
							Description:      descriptions.Get("poller", "schema", "cloudwatch_metrics", "period"),
						},
						"delay": {
							Type:             schema.TypeString,
							ValidateDiagFunc: validateTimeDuration,
							DiffSuppressFunc: diffSuppressTimeDuration,
							Optional:         true,
							Default:          "2m",
							Description:      descriptions.Get("poller", "schema", "cloudwatch_metrics", "delay"),
						},
						"region": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "region"),
						},
						"assume_role_arn": {
							Type:        schema.TypeString,
							Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "assume_role_arn"),
							Required:    true,
						},
						"query": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"namespace": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "namespace"),
									},
									"metric_names": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "metric_names"),
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"dimension": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "dimension"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:     schema.TypeString,
													Required: true,
												},
												"value": {
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"resource_filter": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "description"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"resource_type": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "resource_type"),
												},
												"pattern": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "pattern"),
												},
												"dimension_name": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "dimension_name"),
												},
												"tag_filter": {
													Type:        schema.TypeList,
													Required:    true,
													Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "tag_filter", "description"),
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"key": {
																Type:        schema.TypeString,
																Required:    true,
																Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "tag_filter", "key"),
															},
															"values": {
																Type:        schema.TypeList,
																Optional:    true,
																Elem:        &schema.Schema{Type: schema.TypeString},
																Description: descriptions.Get("poller", "schema", "cloudwatch_metrics", "query", "resource_filter", "tag_filter", "values"),
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
					},
				},
			},
			"aws_snapshot": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: pollerBlockTypes,
				RequiredWith: []string{"interval", "datastream"},
				Description:  descriptions.Get("poller", "schema", "aws_snapshot", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("poller", "schema", "aws_snapshot", "region"),
						},
						"assume_role_arn": {
							Type:        schema.TypeString,
							Description: descriptions.Get("poller", "schema", "aws_snapshot", "assume_role_arn"),
							Required:    true,
						},
						"include_actions": {
							Type:        schema.TypeList,
							Required:    true,
							Description: descriptions.Get("poller", "schema", "aws_snapshot", "include_actions"),
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func newPollerConfig(data *schema.ResourceData) (input *gql.PollerInput, diags diag.Diagnostics) {
	input = &gql.PollerInput{}

	if v, ok := data.GetOk("name"); ok {
		input.Name = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("disabled"); ok {
		input.Disabled = boolPtr(v.(bool))
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
	if v, ok := data.GetOk("skip_external_validation"); ok {
		input.SkipExternalValidation = boolPtr(v.(bool))
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
			Requests:    expandPollerHTTPRequests(data, "http.0.request"),
			Rules:       expandPollerHTTPRules(data, "http.0.rule"),
			Timestamps:  expandPollerHTTPTimestamps(data, "http.0.timestamp"),
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

		if _, ok := data.GetOk("http.0.template"); ok {
			template := expandPollerHTTPRequest(data, "http.0.template.0")
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
	if data.Get("cloudwatch_metrics.#") == 1 {
		m := &gql.PollerCloudWatchMetricsInput{
			Region:        data.Get("cloudwatch_metrics.0.region").(string),
			AssumeRoleArn: data.Get("cloudwatch_metrics.0.assume_role_arn").(string),
			Queries:       expandCloudWatchMetricsQueries(data, "cloudwatch_metrics.0.query"),
		}

		if v, ok := data.GetOk("cloudwatch_metrics.0.period"); ok {
			str := v.(string)
			if period, err := time.ParseDuration(str); err != nil {
				return nil, diag.Errorf("error parsing period: %v", err)
			} else {
				m.Period = types.Int64Scalar(period / time.Second)
			}
		} else {
			// fallback to interval
			m.Period = types.Int64Scalar(int64(*input.Interval) / int64(time.Second))
		}

		if v, ok := data.GetOk("cloudwatch_metrics.0.delay"); ok {
			str := v.(string)
			if delay, err := time.ParseDuration(str); err != nil {
				return nil, diag.Errorf("error parsing delay: %v", err)
			} else {
				m.Delay = types.Int64Scalar(delay / time.Second)
			}
		}

		input.CloudWatchMetricsConfig = m
	}
	if data.Get("aws_snapshot.#") == 1 {
		input.AwsSnapshotConfig = &gql.PollerAWSSnapshotInput{
			Region:         data.Get("aws_snapshot.0.region").(string),
			AssumeRoleArn:  data.Get("aws_snapshot.0.assume_role_arn").(string),
			IncludeActions: makeStrSlice(data.Get("aws_snapshot.0.include_actions").([]interface{})),
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
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
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
	if err := data.Set("kind", poller.Kind); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("disabled", poller.Disabled); err != nil {
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

		timestamp, timestampDiags := flattenPollerHTTPTimestamps(httpConfig.Timestamps)
		diags = append(diags, timestampDiags...)
		if !requestDiags.HasError() {
			ht["timestamp"] = timestamp
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
	if cloudWatchMetricsConfig, ok := config.(*gql.PollerConfigPollerCloudWatchMetricsConfig); ok {
		var queries []map[string]any
		for _, q := range cloudWatchMetricsConfig.Queries {
			v := map[string]any{
				"namespace":    q.Namespace,
				"metric_names": q.MetricNames,
			}

			if len(q.Dimensions) > 0 {
				var dimensions []map[string]any
				for _, d := range q.Dimensions {
					dv := map[string]any{"name": d.Name}
					if d.Value != nil {
						dv["value"] = *d.Value
					}
					dimensions = append(dimensions, dv)
				}
				v["dimension"] = dimensions
			}

			if q.ResourceFilter != nil {
				r := make(map[string]any)
				if s := q.ResourceFilter.ResourceType; s != nil {
					r["resource_type"] = s
				}
				if s := q.ResourceFilter.Pattern; s != nil {
					r["pattern"] = s
				}
				if s := q.ResourceFilter.DimensionName; s != nil {
					r["dimension_name"] = s
				}

				if len(q.ResourceFilter.TagFilters) > 0 {
					var filters []map[string]any
					for _, tf := range q.ResourceFilter.TagFilters {
						filters = append(filters, map[string]any{
							"key":    tf.Key,
							"values": tf.Values,
						})
					}
					r["tag_filter"] = filters
				}
				v["resource_filter"] = []any{r}
			}
			queries = append(queries, v)
		}
		cfg := map[string]any{
			"delay":           time.Duration(int64(cloudWatchMetricsConfig.Delay) * int64(time.Second)).String(),
			"region":          cloudWatchMetricsConfig.Region,
			"assume_role_arn": cloudWatchMetricsConfig.AssumeRoleArn,
			"query":           queries,
		}

		period := time.Duration(int64(cloudWatchMetricsConfig.Period) * int64(time.Second))
		interval := config.GetInterval()
		if interval == nil || time.Duration(*interval) != period {
			cfg["period"] = period.String()
		}

		if err := data.Set("cloudwatch_metrics", []any{cfg}); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if awsSnapshotConfig, ok := config.(*gql.PollerConfigPollerAWSSnapshotConfig); ok {
		cfg := map[string]any{
			"region":          awsSnapshotConfig.Region,
			"assume_role_arn": awsSnapshotConfig.AssumeRoleArn,
		}
		if len(awsSnapshotConfig.IncludeActions) != 0 {
			cfg["include_actions"] = awsSnapshotConfig.IncludeActions
		}
		if err := data.Set("aws_snapshot", []any{cfg}); err != nil {
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
		"url":         req.Url,
		"method":      req.Method,
		"username":    req.Username,
		"password":    req.Password,
		"auth_scheme": req.AuthScheme,
	}

	if req.Body != nil {
		flat["body"] = req.Body
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

func expandPollerHTTPRequests(data *schema.ResourceData, key string) (reqs []gql.PollerHTTPRequestInput) {
	l := data.Get(key).([]interface{})
	if len(l) == 0 {
		return nil
	}

	for i := range l {
		if req := expandPollerHTTPRequest(data, fmt.Sprintf("%s.%d", key, i)); req != nil {
			reqs = append(reqs, *req)
		}
	}

	return
}

func expandPollerHTTPRequest(data *schema.ResourceData, key string) *gql.PollerHTTPRequestInput {
	if v, ok := data.GetOk(key); !ok || v == nil {
		return nil
	}

	headers, _ := json.Marshal(data.Get(key + ".headers").(map[string]interface{}))
	params, _ := json.Marshal(data.Get(key + ".params").(map[string]interface{}))

	parsedHeaders := types.JsonObject(headers)
	parsedParams := types.JsonObject(params)

	req := &gql.PollerHTTPRequestInput{
		Headers: &parsedHeaders,
		Params:  &parsedParams,
	}

	if v, ok := data.GetOk(key + ".url"); ok {
		s := v.(string)
		req.Url = &s
	}

	if v, ok := data.GetOk(key + ".method"); ok {
		s := v.(string)
		req.Method = &s
	}

	if v, ok := data.GetOk(key + ".username"); ok {
		s := v.(string)
		req.Username = &s
	}

	if v, ok := data.GetOk(key + ".password"); ok {
		s := v.(string)
		req.Password = &s
	}

	if v, ok := data.GetOk(key + ".auth_scheme"); ok {
		s := gql.PollerHTTPRequestAuthScheme(toCamel(v.(string)))
		req.AuthScheme = &s
	}

	if v, ok := data.GetOk(key + ".body"); ok {
		s := v.(string)
		req.Body = &s
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

func expandPollerHTTPRules(data *schema.ResourceData, key string) (rules []gql.PollerHTTPRuleInput) {
	l := data.Get(key).([]interface{})
	if len(l) == 0 {
		return nil
	}

	for i := range l {
		if req := expandPollerHTTPRule(data, fmt.Sprintf("%s.%d", key, i)); req != nil {
			rules = append(rules, *req)
		}
	}
	return
}

func expandPollerHTTPRule(data *schema.ResourceData, key string) *gql.PollerHTTPRuleInput {
	if _, ok := data.GetOk(key); !ok {
		return nil
	}

	var match *gql.PollerHTTPRequestInput

	if _, ok := data.GetOk(key + ".match.0"); ok {
		match = expandPollerHTTPRequest(data, key+".match.0")
	}

	var decoder *gql.PollerHTTPDecoderInput
	if _, ok := data.GetOk(key + ".decoder.0"); ok {
		decoder = expandPollerHTTPDecoder(data, key+".decoder.0")
	}

	rule := &gql.PollerHTTPRuleInput{
		Match:   match,
		Decoder: decoder,
	}

	if v, ok := data.GetOk(key + ".follow"); ok {
		s := v.(string)
		rule.Follow = &s
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

func expandPollerHTTPDecoder(data *schema.ResourceData, key string) *gql.PollerHTTPDecoderInput {
	if _, ok := data.GetOk(key); !ok {
		return nil
	}

	decoder := &gql.PollerHTTPDecoderInput{
		Type: data.Get(key + ".type").(string),
	}
	return decoder
}

func flattenPollerHTTPTimestamps(timestamps []gql.PollerConfigPollerHTTPConfigTimestampsPollerHTTPTimestampConfig) (flats []map[string]interface{}, diags diag.Diagnostics) {
	if len(timestamps) == 0 {
		return
	}

	for _, t := range timestamps {
		timestamp, diag := flattenPollerHTTPTimestamp(&t)
		diags = append(diags, diag...)
		if !diag.HasError() {
			flats = append(flats, timestamp)
		}
	}

	return
}

func flattenPollerHTTPTimestamp(timestamp *gql.PollerConfigPollerHTTPConfigTimestampsPollerHTTPTimestampConfig) (flat map[string]interface{}, diags diag.Diagnostics) {
	if timestamp == nil {
		return
	}

	flat = map[string]interface{}{
		"name":     timestamp.Name,
		"source":   timestamp.Source,
		"format":   timestamp.Format,
		"offset":   timestamp.Offset,
		"truncate": timestamp.Truncate,
	}

	return
}

func expandPollerHTTPTimestamps(data *schema.ResourceData, key string) (timestamps []gql.PollerHTTPTimestampInput) {
	l := data.Get(key).([]interface{})
	if len(l) == 0 {
		return nil
	}

	for i := range l {
		if req := expandPollerHTTPTimestamp(data, fmt.Sprintf("%s.%d", key, i)); req != nil {
			timestamps = append(timestamps, *req)
		}
	}
	return
}

func expandPollerHTTPTimestamp(data *schema.ResourceData, key string) *gql.PollerHTTPTimestampInput {
	if _, ok := data.GetOk(key); !ok {
		return nil
	}

	timestamp := &gql.PollerHTTPTimestampInput{}

	if v, ok := data.GetOk(key + ".name"); ok {
		s := v.(string)
		timestamp.Name = &s
	}
	if v, ok := data.GetOk(key + ".source"); ok {
		s := v.(string)
		timestamp.Source = &s
	}
	if v, ok := data.GetOk(key + ".format"); ok {
		s := gql.PollerHTTPTimestampFormatScheme(v.(string))
		timestamp.Format = &s
	}
	if v, ok := data.GetOk(key + ".offset"); ok {
		s := v.(string)
		timestamp.Offset = &s
	}
	if v, ok := data.GetOk(key + ".truncate"); ok {
		s := v.(string)
		timestamp.Truncate = &s
	}

	return timestamp
}

func expandCloudWatchMetricsQueries(data *schema.ResourceData, key string) (qs []gql.PollerCloudWatchMetricsQueryInput) {
	l := data.Get(key).([]interface{})
	if len(l) == 0 {
		return nil
	}

	for i := range l {
		if q := expandCloudWatchMetricsQuery(data, fmt.Sprintf("%s.%d", key, i)); q != nil {
			qs = append(qs, *q)
		}
	}

	return
}

func expandCloudWatchMetricsQuery(data *schema.ResourceData, key string) *gql.PollerCloudWatchMetricsQueryInput {
	if v, ok := data.GetOk(key); !ok || v == nil {
		return nil
	}

	input := &gql.PollerCloudWatchMetricsQueryInput{
		Namespace:      data.Get(key + ".namespace").(string),
		MetricNames:    makeStrSlice(data.Get(key + ".metric_names").([]any)),
		ResourceFilter: expandCloudWatchMetricsResourceFilter(data, key+".resource_filter"),
	}

	for i := range data.Get(key + ".dimension").([]any) {
		input.Dimensions = append(input.Dimensions, gql.PollerCloudWatchMetricsDimensionFilterInput{
			Name: data.Get(fmt.Sprintf("%s.dimension.%d.name", key, i)).(string),
		})
		if v, ok := data.GetOk(fmt.Sprintf("%s.dimension.%d.value", key, i)); ok {
			s := v.(string)
			input.Dimensions[i].Value = &s
		}
	}
	return input
}

func expandCloudWatchMetricsResourceFilter(data *schema.ResourceData, key string) *gql.PollerCloudWatchMetricsResourceFilterInput {
	if v, ok := data.GetOk(key); !ok || v == nil {
		return nil
	}

	var input gql.PollerCloudWatchMetricsResourceFilterInput

	if v, ok := data.GetOk(key + ".0.resource_type"); ok {
		s := v.(string)
		input.ResourceType = &s
	}

	if v, ok := data.GetOk(key + ".0.pattern"); ok {
		s := v.(string)
		input.ResourceType = &s
	}

	if v, ok := data.GetOk(key + ".0.dimension_name"); ok {
		s := v.(string)
		input.DimensionName = &s
	}

	for i := range data.Get(key + ".0.tag_filter").([]any) {
		input.TagFilters = append(input.TagFilters, gql.PollerCloudWatchMetricsTagFilterInput{
			Key:    data.Get(fmt.Sprintf("%s.0.tag_filter.%d.key", key, i)).(string),
			Values: makeStrSlice(data.Get(fmt.Sprintf("%s.0.tag_filter.%d.values", key, i)).([]any)),
		})
	}
	return &input
}
