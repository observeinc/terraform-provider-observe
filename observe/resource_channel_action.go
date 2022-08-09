package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceChannelAction() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelActionCreate,
		ReadContext:   resourceChannelActionRead,
		UpdateContext: resourceChannelActionUpdate,
		DeleteContext: resourceChannelActionDelete,

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
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"rate_limit": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1m",
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"notify_on_close": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"email": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"webhook"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"to": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Required: true,
						},
						"subject": {
							Type:     schema.TypeString,
							Required: true,
						},
						"body": {
							Type:     schema.TypeString,
							Required: true,
						},
						"is_html": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"webhook": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"email"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"method": {
							Type:     schema.TypeString,
							Default:  "POST",
							Optional: true,
						},
						"body": {
							Type:     schema.TypeString,
							Required: true,
						},
						"headers": {
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			"channels": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateOID(oid.TypeChannel),
				},
			},
		},
	}
}

func newChannelActionConfig(data *schema.ResourceData) (input *gql.ActionInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.ActionInput{
		Name: &name,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("rate_limit"); ok {
		rateLimit, _ := types.ParseDurationScalar(v.(string))
		input.RateLimit = rateLimit
	}

	if v, ok := data.GetOk("notify_on_close"); ok {
		input.NotifyOnClose = boolPtr(v.(bool))
	}

	if _, ok := data.GetOk("webhook"); ok {
		webhook := expandWebhookConfig(data.Get("webhook.0").(map[string]interface{}))
		input.Webhook = &webhook
	}

	if _, ok := data.GetOk("email"); ok {
		email := expandEmailConfig(data.Get("email.0").(map[string]interface{}))
		input.Email = &email
	}

	return
}

func channels(data *schema.ResourceData) (channels []string, diags diag.Diagnostics) {
	if s, ok := data.GetOk("channels"); ok {
		for _, v := range s.(*schema.Set).List() {
			id, _ := oid.NewOID(v.(string))
			channels = append(channels, id.Id)
		}
	}
	return
}

func expandWebhookConfig(data map[string]interface{}) gql.WebhookActionInput {
	var (
		urlTemplate  = data["url"].(string)
		method       = data["method"].(string)
		bodyTemplate = data["body"].(string)
	)
	config := gql.WebhookActionInput{
		UrlTemplate:  &urlTemplate,
		Method:       &method,
		BodyTemplate: &bodyTemplate,
	}

	if s, ok := data["headers"].(map[string]interface{}); ok {
		config.Headers = make([]gql.WebhookHeaderInput, 0)
		for k, v := range s {
			config.Headers = append(config.Headers, gql.WebhookHeaderInput{
				Header:        k,
				ValueTemplate: v.(string),
			})
		}
	}

	return config
}

func flattenWebhook(webhook *gql.ChannelActionWebhookAction) []map[string]interface{} {
	headers := make(map[string]string)
	for _, header := range webhook.Headers {
		headers[header.Header] = header.ValueTemplate
	}
	data := map[string]interface{}{
		"url":     webhook.UrlTemplate,
		"method":  webhook.Method,
		"body":    webhook.BodyTemplate,
		"headers": headers,
	}
	return []map[string]interface{}{data}
}

func expandEmailConfig(data map[string]interface{}) gql.EmailActionInput {
	var (
		subjectTemplate = data["subject"].(string)
		bodyTemplate    = data["body"].(string)
		isHtml          = data["is_html"].(bool)
	)
	config := gql.EmailActionInput{
		SubjectTemplate: &subjectTemplate,
		BodyTemplate:    &bodyTemplate,
		IsHtml:          &isHtml,
	}

	for _, v := range data["to"].([]interface{}) {
		config.TargetAddresses = append(config.TargetAddresses, v.(string))
	}
	return config
}

func flattenEmail(email *gql.ChannelActionEmailAction) []map[string]interface{} {
	data := map[string]interface{}{
		"to":      email.TargetAddresses,
		"subject": email.SubjectTemplate,
		"body":    email.BodyTemplate,
		"is_html": email.IsHtml,
	}
	return []map[string]interface{}{data}
}

func resourceChannelActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelActionConfig(data)
	if diags.HasError() {
		return diags
	}
	channels, diags := channels(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateChannelAction(ctx, id.Id, config, channels)
	if err != nil {
		return diag.Errorf("failed to create channel action: %s", err.Error())
	}

	data.SetId((*result).GetId())
	return append(diags, resourceChannelActionRead(ctx, data, meta)...)
}

func resourceChannelActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelActionConfig(data)
	if diags.HasError() {
		return diags
	}
	channels, diags := channels(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateChannelAction(ctx, data.Id(), config, channels)
	if err != nil {
		return diag.Errorf("failed to update channel action: %s", err.Error())
	}

	return append(diags, resourceChannelActionRead(ctx, data, meta)...)
}

func resourceChannelActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	channelActionPtr, err := client.GetChannelAction(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read channel action: %s", err.Error())
	}
	channelAction := *channelActionPtr

	if err := data.Set("workspace", oid.WorkspaceOid(channelAction.GetWorkspaceId()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", channelAction.GetName()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", channelAction.GetIconUrl()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", channelAction.GetDescription()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if channelAction.GetRateLimit() != 0 {
		if err := data.Set("rate_limit", channelAction.GetRateLimit().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("notify_on_close", channelAction.GetNotifyOnClose()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if webhook, ok := channelAction.(*gql.ChannelActionWebhookAction); ok {
		if err := data.Set("webhook", flattenWebhook(webhook)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if email, ok := channelAction.(*gql.ChannelActionEmailAction); ok {
		if err := data.Set("email", flattenEmail(email)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	channels := make([]string, 0)
	for _, channel := range channelAction.GetChannels() {
		channels = append(channels, oid.ChannelOid(channel.Id).String())
	}
	if err := data.Set("channels", channels); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", gql.ChannelActionOid(channelAction).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourceChannelActionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteChannelAction(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete action: %s", err.Error())
	}
	return diags
}
