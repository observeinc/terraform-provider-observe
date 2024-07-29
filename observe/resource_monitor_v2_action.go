package observe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceMonitorV2Action() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2Create,
		ReadContext:   resourceMonitorV2Read,
		UpdateContext: resourceMonitorV2Update,
		DeleteContext: resourceMonitorV2Delete,
		Schema: map[string]*schema.Schema{
			// needed as input to CreateMonitorV2Action
			"workspace": { // ObjectId!
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			// fields of MonitorV2ActionInput
			"type": { // MonitorV2ActionType!
				Type:             schema.TypeString,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2ActionTypes),
				Required:         true,
			},
			"email": { // MonitorV2EmailDestinationInput
				Type:         schema.TypeList,
				MaxItems:     1,
				Optional:     true,
				ExactlyOneOf: []string{"email", "webhook"},
				Elem:         monitorV2EmailActionInput(),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:         schema.TypeList,
				MaxItems:     1,
				Optional:     true,
				ExactlyOneOf: []string{"email", "webhook"},
				Elem:         monitorV2WebhookActionInput(),
			},
			"name": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			// end of monitorV2ActionInput
			"oid": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2EmailActionInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": { // String
				Type:     schema.TypeString,
				Required: true,
			},
			"body": { // String
				Type:     schema.TypeString,
				Required: true,
			},
			"fragments": { // JsonObject
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Optional:         true,
			},
		},
	}
}

func monitorV2WebhookActionInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"headers": { // [MonitorV2WebhookHeaderInput!]
				Type:     schema.TypeList,
				Optional: true,
				Elem:     monitorV2WebhookHeaderInput(),
			},
			"body": { // String
				Type:     schema.TypeString,
				Required: true,
			},
			"fragments": { // JsonObject
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Optional:         true,
			},
		},
	}
}

func monitorV2WebhookHeaderInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"header": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"value": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceMonitorV2ActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}

	workspaceID, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitorV2Action(ctx, workspaceID.Id, input)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2Action(ctx, data.Id(), input)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2ActionCreate(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorV2Action(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor action: %s", err.Error())
	}
	return diags
}

func resourceMonitorV2ActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	action, err := client.GetMonitorV2Action(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitorv2 action: %s", err.Error())
	}

	if err := data.Set("workspace", oid.WorkspaceOid(action.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("type", toSnake(string(action.GetType()))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", action.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", action.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if action.Email != nil {
		if err := data.Set("email", monitorV2FlattenEmailAction(*action.Email)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if action.Webhook != nil {
		if err := data.Set("webhook", monitorV2FlattenWebhookAction(*action.Webhook)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if action.IconUrl != nil {
		if err := data.Set("icon_url", *action.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if action.Description != nil {
		if err := data.Set("description", *action.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func monitorV2FlattenEmailAction(gqlEmail gql.MonitorV2EmailAction) interface{} {
	email := make(map[string]interface{})
	if gqlEmail.Body != nil {
		email["body"] = *gqlEmail.Body
	}
	if gqlEmail.Fragments != nil {
		email["fragments"] = gqlEmail.Fragments.String()
	}
	if gqlEmail.Subject != nil {
		email["subject"] = *gqlEmail.Subject
	}
	return email
}

func monitorV2FlattenWebhookAction(gqlWebhook gql.MonitorV2WebhookAction) interface{} {
	webhook := make(map[string]interface{})
	if len(gqlWebhook.Headers) > 0 {
		webhook["headers"] = monitorV2FlattenWebhookHeaders(gqlWebhook.Headers)
	}
	if gqlWebhook.Fragments != nil {
		webhook["fragments"] = gqlWebhook.Fragments.String()
	}
	if gqlWebhook.Body != nil {
		webhook["body"] = *gqlWebhook.Body
	}
	return webhook
}

func monitorV2FlattenWebhookHeaders(gqlHeaders []gql.MonitorV2WebhookHeader) []interface{} {
	var headers []interface{}
	for _, gqlHeader := range gqlHeaders {
		headers = append(headers, monitorV2FlattenWebhookHeader(gqlHeader))
	}
	return headers
}

func monitorV2FlattenWebhookHeader(gqlHeader gql.MonitorV2WebhookHeader) interface{} {
	header := map[string]interface{}{
		"header": gqlHeader.Header,
		"value":  gqlHeader.Value,
	}
	return header
}

func newMonitorV2ActionInput(data *schema.ResourceData) (input *gql.MonitorV2ActionInput, diags diag.Diagnostics) {
	// required
	actionType := toCamel(data.Get("type").(string))
	name := data.Get("name").(string)

	// instantiation
	inlineVal := false
	input = &gql.MonitorV2ActionInput{
		Type:   meta.MonitorV2ActionType(actionType),
		Name:   name,
		Inline: &inlineVal, // we are not currently allowing inline actions
	}

	// optionals
	if _, ok := data.GetOk("email"); ok {
		email, diags := newMonitorV2EmailActionInput(data, "email.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Email = email
	}
	if _, ok := data.GetOk("webhook"); ok {
		webhook, diags := newMonitorV2WebhookActionInput(data, "webhook.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Webhook = webhook
	}
	if v, ok := data.GetOk("icon_url"); ok {
		iconURL := v.(string)
		input.IconUrl = &iconURL
	}
	if v, ok := data.GetOk("description"); ok {
		description := v.(string)
		input.Description = &description
	}

	return nil, diags
}

func newMonitorV2EmailActionInput(data *schema.ResourceData, path string) (email *gql.MonitorV2EmailActionInput, diags diag.Diagnostics) {
	// required
	subject := data.Get(fmt.Sprintf("%ssubject", path)).(string)
	email.Subject = &subject
	body := data.Get(fmt.Sprintf("%sbody", path)).(string)
	email.Body = &body

	// instantiation
	email = &gql.MonitorV2EmailActionInput{}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		email.Fragments = types.JsonObject(v.(string)).Ptr()
	}

	return email, diags
}

func newMonitorV2WebhookActionInput(data *schema.ResourceData, path string) (webhook *gql.MonitorV2WebhookActionInput, diags diag.Diagnostics) {
	// required
	body := data.Get(fmt.Sprintf("%sbody", path)).(string)
	webhook.Body = &body

	// instantiation
	webhook = &gql.MonitorV2WebhookActionInput{}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%sheaders", path)); ok {
		webhook.Headers = make([]gql.MonitorV2WebhookHeaderInput, 0)
		for i := range data.Get(fmt.Sprintf("%sheaders", path)).([]interface{}) {
			header, diags := newMonitorV2WebhookHeaderInput(data, fmt.Sprintf("%sheaders.%d.", path, i))
			if diags.HasError() {
				return nil, diags
			}
			webhook.Headers = append(webhook.Headers, *header)
		}
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		fragmentsStr := v.(string)
		err := json.Unmarshal([]byte(fragmentsStr), webhook.Fragments)
		if err != nil {
			return nil, diag.Errorf(err.Error())
		}
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		webhook.Fragments = types.JsonObject(v.(string)).Ptr()
	}

	return webhook, diags
}

func newMonitorV2WebhookHeaderInput(data *schema.ResourceData, path string) (header *gql.MonitorV2WebhookHeaderInput, diags diag.Diagnostics) {
	// required
	headerStr := data.Get(fmt.Sprintf("%sheader", path)).(string)
	valueStr := data.Get(fmt.Sprintf("%svalue", path)).(string)

	// instantiation
	header = &gql.MonitorV2WebhookHeaderInput{
		Header: headerStr,
		Value:  valueStr,
	}

	return header, diags
}
