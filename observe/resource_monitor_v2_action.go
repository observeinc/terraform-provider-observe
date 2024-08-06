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

/************************************************************
*                                                           *
*   FOR WHOEVER GETS PUT ON THIS TASK AFTER I LEAVE:        *
*                                                           *
*   This file was written when the API used to require      *
*   separate calls for creating a monv2 action and a        *
*   destination. This is why, for example, the function     *
*   resourceMonitorV2DestinationCreate calls 2 create       *
*   API calls (one to make the action, another for dest).   *
*   If you're reading this message with the intent of       *
*   modifying this file, the API has probably changed       *
*   so that a shared action and inlined destination can     *
*   be edited in a single API call, which is likely         *
*   what you were asked to change about this code.          *
*                                                           *
*   If possible, please do NOT remove any params from the   *
*   existing schema or add any new required params. I       *
*   tried to write the schema so that it would map onto     *
*   the new API relatively cleanly. You probably won't      *
*   need to change the schema, but you'll likely need to    *
*   change how the variables read from said schema are      *
*   arranged into the inputs fed to the API call.           *
*                                                           *
*   After making the changes, please delete this comment.   *
*                                                           *
*   Thanks! :)                                              *
*   - Owen                                                  *
*                                                           *
***********************************************************/

func resourceMonitorV2Action() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2ActionCreate,
		ReadContext:   resourceMonitorV2ActionRead,
		UpdateContext: resourceMonitorV2ActionUpdate,
		DeleteContext: resourceMonitorV2ActionDelete,
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
			"destination": { //
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem:     resourceMonitorV2Destination(),
			},
		},
	}
}

func monitorV2EmailActionInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": { // String
				Type:     schema.TypeString,
				Optional: true,
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
				Optional: true,
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

func resourceMonitorV2Destination() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"email": { // MonitorV2WebhookDestinationInput
				Type:         schema.TypeList,
				Optional:     true,
				ExactlyOneOf: []string{"destination.0.email", "destination.0.webhook"},
				Elem:         monitorV2EmailDestinationResource(),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:         schema.TypeList,
				Optional:     true,
				ExactlyOneOf: []string{"destination.0.email", "destination.0.webhook"},
				Elem:         monitorV2WebhookDestinationResource(),
			},
			"description": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			// ^^^ end of input
			"oid": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2WebhookDestinationResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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

func monitorV2EmailDestinationResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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

func resourceMonitorV2ActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	actInput, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}
	dstInput, diags := newMonitorV2DestinationInput(actInput)
	if diags.HasError() {
		return diags
	}

	workspaceID, _ := oid.NewOID(data.Get("workspace").(string))
	actResult, err := client.CreateMonitorV2Action(ctx, workspaceID.Id, actInput)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	dstResult, err := client.CreateMonitorV2Destination(ctx, workspaceID.Id, dstInput)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}
	if err := data.Set("destination", monitorV2FlattenDestination(*dstResult)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	dstLinks := []gql.ActionDestinationLinkInput{
		{
			DestinationID: dstResult.Id,
		},
	}
	_, err = client.SaveActionWithDestinationLinks(ctx, actResult.Id, dstLinks)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId(actResult.Id)
	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	actId := data.Id()
	var dstId string

	if v, ok := data.GetOk("destination.0.oid"); ok {
		dstOID, err := oid.NewOID(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		dstId = dstOID.Id
	} else {
		return diag.Errorf("no destination id found")
	}

	actInput, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}
	dstInput, diags := newMonitorV2DestinationInput(actInput)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2Action(ctx, actId, actInput)
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

	_, err = client.UpdateMonitorV2Destination(ctx, dstId, dstInput)
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

	dstLinks := []gql.ActionDestinationLinkInput{
		{
			DestinationID: dstId,
		},
	}
	_, err = client.Meta.SaveActionWithDestinationLinks(ctx, actId, dstLinks)
	if err != nil {
		return diag.FromErr(err)
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

	if len(action.DestinationLinks) < 1 {
		return diag.Errorf("no destination id found")
	}

	dstId := action.DestinationLinks[0].DestinationID
	dst, err := client.GetMonitorV2Destination(ctx, dstId)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read monitorv2 action: %s", err.Error())
	}

	if err := data.Set("destination", monitorV2FlattenDestination(*dst)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
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
	if gqlEmail.Body != nil {
		email["body"] = *gqlEmail.Body
	}
	if gqlEmail.Fragments != nil {
		email["fragments"] = gqlEmail.Fragments.String()
	}
	if gqlEmail.Subject != nil {
		email["subject"] = *gqlEmail.Subject
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
	if gqlWebhook.Body != nil {
		webhook["body"] = *gqlWebhook.Body
	}
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
	if v, ok := data.GetOk("description"); ok {
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
		subject := v.(string)
		email.Subject = &subject
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		body := v.(string)
		email.Body = &body
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		email.Fragments = types.JsonObject(v.(string)).Ptr()
	}

	return email, diags
}

func newMonitorV2WebhookActionInput(data *schema.ResourceData, path string) (webhook *gql.MonitorV2WebhookActionInput, diags diag.Diagnostics) {

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
		webhook.Fragments = types.JsonObject(v.(string)).Ptr()
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		body := v.(string)
		webhook.Body = &body
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
