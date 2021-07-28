package observe

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
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
			"rate_limit": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1m",
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
			"email": &schema.Schema{
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
			"webhook": &schema.Schema{
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
					ValidateDiagFunc: validateOID(observe.TypeChannel),
				},
			},
		},
	}
}

func newChannelActionConfig(data *schema.ResourceData) (config *observe.ChannelActionConfig, diags diag.Diagnostics) {

	config = &observe.ChannelActionConfig{
		Name: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		s := v.(string)
		config.IconURL = &s
	}

	if v, ok := data.GetOk("description"); ok {
		s := v.(string)
		config.Description = &s
	}

	if v, ok := data.GetOk("rate_limit"); ok {
		// we already validated in schema
		t, _ := time.ParseDuration(v.(string))
		config.RateLimit = &t
	}

	if _, ok := data.GetOk("webhook"); ok {
		config.Webhook = expandWebhookConfig(data.Get("webhook.0").(map[string]interface{}))
	}

	if _, ok := data.GetOk("email"); ok {
		config.Email = expandEmailConfig(data.Get("email.0").(map[string]interface{}))
	}

	if s, ok := data.GetOk("channels"); ok {
		for _, v := range s.(*schema.Set).List() {
			oid, _ := observe.NewOID(v.(string))
			config.Channels = append(config.Channels, oid)
		}
	}

	return
}

func expandWebhookConfig(data map[string]interface{}) *observe.WebhookChannelActionConfig {
	config := &observe.WebhookChannelActionConfig{
		URL:    data["url"].(string),
		Method: data["method"].(string),
		Body:   data["body"].(string),
	}

	if s, ok := data["headers"].(map[string]interface{}); ok {
		config.Headers = make(map[string]string)
		for k, v := range s {
			config.Headers[k] = v.(string)
		}
	}

	return config
}

func flattenWebhookConfig(config *observe.WebhookChannelActionConfig) map[string]interface{} {
	data := map[string]interface{}{
		"url":     config.URL,
		"method":  config.Method,
		"body":    config.Body,
		"headers": config.Headers,
	}
	return data
}

func expandEmailConfig(data map[string]interface{}) *observe.EmailChannelActionConfig {
	config := &observe.EmailChannelActionConfig{
		Subject: data["subject"].(string),
		Body:    data["body"].(string),
		IsHTML:  data["is_html"].(bool),
	}

	for _, v := range data["to"].([]interface{}) {
		config.To = append(config.To, v.(string))
	}
	return config
}

func flattenEmailConfig(config *observe.EmailChannelActionConfig) map[string]interface{} {
	data := map[string]interface{}{
		"to":      config.To,
		"subject": config.Subject,
		"body":    config.Body,
		"isHtml":  config.IsHTML,
	}
	return data
}

func resourceChannelActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelActionConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateChannelAction(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create channel action: %s", err.Error())
	}

	data.SetId(result.ID)
	return append(diags, resourceChannelActionRead(ctx, data, meta)...)
}

func resourceChannelActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelActionConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateChannelAction(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update channel action: %s", err.Error())
	}

	return append(diags, resourceChannelActionRead(ctx, data, meta)...)
}

func resourceChannelActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	channelAction, err := client.GetChannelAction(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read channel action: %s", err.Error())
	}

	workspaceOID := observe.OID{
		Type: observe.TypeWorkspace,
		ID:   channelAction.WorkspaceID,
	}

	if err := data.Set("workspace", workspaceOID.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", channelAction.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", channelAction.Config.IconURL); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", channelAction.Config.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if channelAction.Config.RateLimit != nil {
		if err := data.Set("rate_limit", channelAction.Config.RateLimit.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if channelAction.Config.Webhook != nil {
		if err := data.Set("webhook", flattenWebhookConfig(channelAction.Config.Webhook)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if channelAction.Config.Email != nil {
		if err := data.Set("email", flattenEmailConfig(channelAction.Config.Email)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("channels", toListOfStrings(channelAction.Config.Channels)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", channelAction.OID().String()); err != nil {
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
