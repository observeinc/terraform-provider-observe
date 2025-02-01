package observe

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceMonitorAction() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("monitor_action", "description"),
		CreateContext: resourceMonitorActionCreate,
		ReadContext:   resourceMonitorActionRead,
		UpdateContext: resourceMonitorActionUpdate,
		DeleteContext: resourceMonitorActionDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("monitor_action", "schema", "name"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("monitor_action", "schema", "description"),
			},
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"rate_limit": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "10m",
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
				Description:      descriptions.Get("monitor_action", "schema", "rate_limit"),
			},
			"notify_on_close": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: descriptions.Get("monitor_action", "schema", "notify_on_close"),
			},
			"email": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"email", "webhook"},
				Description:  descriptions.Get("monitor_action", "schema", "email", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_addresses": {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: descriptions.Get("monitor_action", "schema", "email", "target_addresses"),
						},
						"subject_template": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitor_action", "schema", "email", "subject_template"),
						},
						"body_template": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitor_action", "schema", "email", "body_template"),
						},
						"is_html": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: descriptions.Get("monitor_action", "schema", "email", "is_html"),
						},
					},
				},
			},
			"webhook": {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{"email", "webhook"},
				Description:  descriptions.Get("monitor_action", "schema", "webhook", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url_template": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitor_action", "schema", "webhook", "url_template"),
						},
						"method": {
							Type:        schema.TypeString,
							Default:     "POST",
							Optional:    true,
							Description: descriptions.Get("monitor_action", "schema", "webhook", "method"),
						},
						"body_template": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("monitor_action", "schema", "webhook", "body_template"),
						},
						"headers": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: descriptions.Get("monitor_action", "schema", "webhook", "headers"),
						},
					},
				},
			},
		},
	}
}

func newMonitorActionConfig(data *schema.ResourceData) (input *gql.MonitorActionInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	id, err := oid.NewOID(data.Get("workspace").(string))
	if err != nil {
		return nil, diag.Errorf("failed to get monitor action workspace id: %s", err.Error())
	}
	input = &gql.MonitorActionInput{
		Name:        name,
		WorkspaceId: id.Id,
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
		input.NotifyOnClose = v.(bool)
	}

	if _, ok := data.GetOk("webhook"); ok {
		webhook := expandMonitorActionWebhookConfig(data.Get("webhook.0").(map[string]interface{}))
		input.Webhook = &webhook
	}

	if _, ok := data.GetOk("email"); ok {
		email := expandMonitorActionEmailConfig(data.Get("email.0").(map[string]interface{}))
		input.Email = &email
	}

	return
}

func expandMonitorActionWebhookConfig(data map[string]interface{}) gql.WebhookActionInput {
	var (
		urlTemplate  = data["url_template"].(string)
		method       = data["method"].(string)
		bodyTemplate = data["body_template"].(string)
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

func flattenMonitorActionWebhook(webhook *gql.MonitorActionWebhookAction) []map[string]interface{} {
	headers := make(map[string]string)
	for _, header := range webhook.Headers {
		headers[header.Header] = header.ValueTemplate
	}
	data := map[string]interface{}{
		"url_template":  webhook.UrlTemplate,
		"method":        webhook.Method,
		"body_template": webhook.BodyTemplate,
		"headers":       headers,
	}
	return []map[string]interface{}{data}
}

func expandMonitorActionEmailConfig(data map[string]interface{}) gql.EmailActionInput {
	var (
		subjectTemplate = data["subject_template"].(string)
		bodyTemplate    = data["body_template"].(string)
		isHtml          = data["is_html"].(bool)
	)
	config := gql.EmailActionInput{
		SubjectTemplate: &subjectTemplate,
		BodyTemplate:    &bodyTemplate,
		IsHtml:          &isHtml,
	}

	for _, v := range data["target_addresses"].([]interface{}) {
		config.TargetAddresses = append(config.TargetAddresses, v.(string))
	}
	return config
}

func flattenMonitorActionEmail(email *gql.MonitorActionEmailAction) []map[string]interface{} {
	data := map[string]interface{}{
		"target_addresses": email.TargetAddresses,
		"subject_template": email.SubjectTemplate,
		"body_template":    email.BodyTemplate,
		"is_html":          email.IsHtml,
	}
	return []map[string]interface{}{data}
}

func resourceMonitorActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorActionConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateMonitorAction(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId((*result).GetId())
	return append(diags, resourceMonitorActionRead(ctx, data, meta)...)
}

func resourceMonitorActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newMonitorActionConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorAction(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update monitor action: %s", err.Error())
	}

	return append(diags, resourceMonitorActionRead(ctx, data, meta)...)
}

func resourceMonitorActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	monitorActionPtr, err := client.GetMonitorAction(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitor action: %s", err.Error())
	}
	monitorAction := *monitorActionPtr

	if err := data.Set("workspace", oid.WorkspaceOid(monitorAction.GetWorkspaceId()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", monitorAction.GetName()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", monitorAction.GetIconUrl()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", monitorAction.GetDescription()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if monitorAction.GetRateLimit() != 0 {
		if err := data.Set("rate_limit", monitorAction.GetRateLimit().String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("notify_on_close", monitorAction.GetNotifyOnClose()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if webhook, ok := monitorAction.(*gql.MonitorActionWebhookAction); ok {
		if err := data.Set("webhook", flattenMonitorActionWebhook(webhook)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if email, ok := monitorAction.(*gql.MonitorActionEmailAction); ok {
		if err := data.Set("email", flattenMonitorActionEmail(email)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", gql.MonitorActionOid(monitorAction).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourceMonitorActionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorAction(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor: %s", err.Error())
	}
	return diags
}
