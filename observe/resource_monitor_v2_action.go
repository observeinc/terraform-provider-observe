package observe

import (
	"context"
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
		CreateContext: resourceMonitorV2ActionCreate,
		ReadContext:   resourceMonitorV2ActionRead,
		UpdateContext: resourceMonitorV2ActionUpdate,
		DeleteContext: resourceMonitorV2ActionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
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
				Optional: true,
			},
			"fragments": { // JsonObject
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Optional:         true,
			},
			"users": { // [UserId!]
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateOID(oid.TypeUser),
				},
			},
			"addresses": { // [String!]
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
			"url": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"method": { // MonitorV2HttpType!
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2HttpTypes),
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

	actInput, diags := newMonitorV2ActionInput("", data)
	if diags.HasError() {
		return diags
	}

	workspaceID, _ := oid.NewOID(data.Get("workspace").(string))
	actResult, err := client.CreateMonitorV2Action(ctx, workspaceID.Id, actInput)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId(actResult.Id)
	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	actInput, diags := newMonitorV2ActionInput("", data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2Action(ctx, data.Id(), actInput)
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

	actId := data.Id()
	action, err := client.GetMonitorV2Action(ctx, actId)
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

	if action.Description != nil {
		if err := data.Set("description", *action.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func monitorV2FlattenEmailAction(gqlEmail gql.MonitorV2EmailAction) []interface{} {
	email := make(map[string]interface{})
	email["subject"] = gqlEmail.Subject
	if gqlEmail.Body != nil {
		email["body"] = *gqlEmail.Body
	}

	if gqlEmail.Fragments != nil {
		email["fragments"] = gqlEmail.Fragments.String()
	}

	if len(gqlEmail.Addresses) > 0 {
		addrs := make([]string, 0)
		for _, addr := range gqlEmail.Addresses {
			addrs = append(addrs, addr)
		}
		email["addresses"] = addrs
	}
	if len(gqlEmail.Users) > 0 {
		userOIDStrs := make([]string, 0)
		for _, uid := range gqlEmail.Users {
			userOID := oid.UserOid(uid)
			userOIDStrs = append(userOIDStrs, userOID.String())
		}
		email["users"] = userOIDStrs
	}
	return []interface{}{email}
}

func monitorV2FlattenWebhookAction(gqlWebhook gql.MonitorV2WebhookAction) []interface{} {
	webhook := make(map[string]interface{})
	if len(gqlWebhook.Headers) > 0 {
		webhook["headers"] = monitorV2FlattenWebhookHeaders(gqlWebhook.Headers)
	}
	if gqlWebhook.Fragments != nil {
		webhook["fragments"] = gqlWebhook.Fragments.String()
	}
	webhook["url"] = gqlWebhook.Url
	webhook["method"] = toSnake(string(gqlWebhook.Method))
	webhook["body"] = gqlWebhook.Body

	return []interface{}{webhook}
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

func newMonitorV2ActionInput(path string, data *schema.ResourceData) (input *gql.MonitorV2ActionInput, diags diag.Diagnostics) {
	// required
	actionType := toCamel(data.Get(fmt.Sprintf("%stype", path)).(string))

	// instantiation
	var inline = false // default behavior is that explicit action creation is for shared (non-inline) actions only
	input = &gql.MonitorV2ActionInput{
		Type:   meta.MonitorV2ActionType(actionType),
		Inline: &inline,
	}

	if name, ok := data.GetOk(fmt.Sprintf("%sname", path)); ok {
		input.Name = name.(string)
	}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%semail", path)); ok {
		email, diags := newMonitorV2EmailActionInput(data, fmt.Sprintf("%semail.0.", path))
		if diags.HasError() {
			return nil, diags
		}
		input.Email = email
	}
	if _, ok := data.GetOk(fmt.Sprintf("%swebhook", path)); ok {
		webhook, diags := newMonitorV2WebhookActionInput(data, fmt.Sprintf("%swebhook.0.", path))
		if diags.HasError() {
			return nil, diags
		}
		input.Webhook = webhook
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sdescription", path)); ok {
		description := v.(string)
		input.Description = &description
	}

	return input, diags
}

func newMonitorV2EmailActionInput(data *schema.ResourceData, path string) (email *gql.MonitorV2EmailActionInput, diags diag.Diagnostics) {
	// instantiation
	email = &gql.MonitorV2EmailActionInput{}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%ssubject", path)); ok {
		email.Subject = v.(string)
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		body := v.(string)
		email.Body = &body
	} else {
		// body must be empty string, NOT JSON NULL
		emptyString := ""
		email.Body = &emptyString
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		email.Fragments = types.JsonObject(v.(string)).Ptr()
	}
	if _, ok := data.GetOk(fmt.Sprintf("%susers", path)); ok {
		email.Users = make([]types.UserIdScalar, 0)
		for i := range data.Get(fmt.Sprintf("%susers", path)).([]interface{}) {
			userOID, err := oid.NewOID(data.Get(fmt.Sprintf("%susers.%d", path, i)).(string))
			if err != nil {
				return nil, diag.FromErr(err)
			}
			uid, err := types.StringToUserIdScalar(userOID.Id)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			email.Users = append(email.Users, uid)
		}
	}
	if _, ok := data.GetOk(fmt.Sprintf("%saddresses", path)); ok {
		email.Addresses = make([]string, 0)
		for i := range data.Get(fmt.Sprintf("%saddresses", path)).([]interface{}) {
			addr := data.Get(fmt.Sprintf("%saddresses.%d", path, i)).(string)
			email.Addresses = append(email.Addresses, addr)
		}
	}

	return email, diags
}

func newMonitorV2WebhookActionInput(data *schema.ResourceData, path string) (webhook *gql.MonitorV2WebhookActionInput, diags diag.Diagnostics) {
	url := data.Get(fmt.Sprintf("%surl", path)).(string)
	method := gql.MonitorV2HttpType(toCamel(data.Get(fmt.Sprintf("%smethod", path)).(string)))

	// instantiation
	webhook = &gql.MonitorV2WebhookActionInput{
		Url:    url,
		Method: method,
	}

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
		webhook.Fragments = types.JsonObject(v.(string)).Ptr()
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		webhook.Body = v.(string)
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
