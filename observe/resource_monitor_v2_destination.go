package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
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

func resourceMonitorV2Destination() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2DestinationCreate,
		ReadContext:   resourceMonitorV2DestinationRead,
		UpdateContext: resourceMonitorV2DestinationUpdate,
		DeleteContext: resourceMonitorV2DestinationDelete,
		Schema: map[string]*schema.Schema{
			"workspace": { // ?
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"action": { // associated action
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeMonitorV2Action),
			},
			"type": { // MonitorV2ActionType!
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2ActionTypes),
			},
			"email": { // MonitorV2WebhookDestinationInput
				Type:         schema.TypeList,
				Optional:     true,
				ExactlyOneOf: []string{"email", "webhook"},
				Elem:         monitorV2EmailDestinationResource(),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:         schema.TypeList,
				Optional:     true,
				ExactlyOneOf: []string{"email", "webhook"},
				Elem:         monitorV2WebhookDestinationResource(),
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
					ValidateDiagFunc: validateUID(),
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

func resourceMonitorV2DestinationCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2DestinationInput(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateMonitorV2Destination(ctx, id.Id, input)
	if err != nil {
		return diag.FromErr(err)
	}

	// update the link between action and destination
	actOID, _ := oid.NewOID(data.Get("action").(string))
	dstLinks := []gql.ActionDestinationLinkInput{
		{
			DestinationID: result.Id,
		},
	}
	_, err = client.Meta.SaveActionWithDestinationLinks(ctx, actOID.Id, dstLinks)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(result.Id)
	return append(diags, resourceMonitorV2DestinationRead(ctx, data, meta)...)
}

func resourceMonitorV2DestinationUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2DestinationInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2Destination(ctx, data.Id(), input)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2DestinationCreate(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.FromErr(err)
	}

	// update the link between action and destination
	actOID, _ := oid.NewOID(data.Get("action").(string))
	dstLinks := []gql.ActionDestinationLinkInput{
		{
			DestinationID: data.Id(),
		},
	}
	_, err = client.Meta.SaveActionWithDestinationLinks(ctx, actOID.Id, dstLinks)
	if err != nil {
		return diag.FromErr(err)
	}

	return append(diags, resourceMonitorV2DestinationRead(ctx, data, meta)...)
}

func resourceMonitorV2DestinationRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	dest, err := client.GetMonitorV2Destination(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// required
	if err := data.Set("workspace", oid.WorkspaceOid(dest.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", dest.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", dest.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("type", toSnake(string(dest.Type))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// optional
	if dest.Description != nil {
		if err := data.Set("description", *dest.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if dest.Email != nil {
		if err := data.Set("email", monitorV2FlattenEmailDestination(*dest.Email)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if dest.IconUrl != nil {
		if err := data.Set("icon_url", *dest.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if dest.Webhook != nil {
		if err := data.Set("webhook", monitorv2FlattenWebhookDestination(*dest.Webhook)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func resourceMonitorV2DestinationDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorV2Destination(ctx, data.Id()); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func monitorV2FlattenEmailDestination(gqlEmail gql.MonitorV2EmailDestination) []interface{} {
	email := make(map[string]interface{})
	if len(gqlEmail.Addresses) > 0 {
		addrs := make([]string, 0)
		for _, addr := range gqlEmail.Addresses {
			addrs = append(addrs, addr)
		}
		email["addresses"] = addrs
	}
	if len(gqlEmail.Users) > 0 {
		uidStrs := make([]string, 0)
		for _, uid := range gqlEmail.Users {
			uidStrs = append(uidStrs, uid.String())
		}
		email["users"] = uidStrs
	}
	return []interface{}{email}
}

func monitorv2FlattenWebhookDestination(gqlWebhook gql.MonitorV2WebhookDestination) []interface{} {
	webhook := map[string]interface{}{
		"url":    gqlWebhook.Url,
		"method": toSnake(string(gqlWebhook.Method)),
	}
	return []interface{}{webhook}
}

func newMonitorV2DestinationInput(data *schema.ResourceData) (input *gql.MonitorV2DestinationInput, diags diag.Diagnostics) {
	// required
	actionType := gql.MonitorV2ActionType(toCamel(data.Get("type").(string)))
	name := data.Get("name").(string)

	// instantiation
	inlineVal := true // we are currently only allowing destinations to be inlined
	input = &gql.MonitorV2DestinationInput{
		Type:   actionType,
		Name:   name,
		Inline: &inlineVal,
	}

	// optionals
	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}
	if _, ok := data.GetOk("email"); ok {
		email, diags := newMonitorV2EmailDestinationInput(data, "email.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Email = email
	}
	if _, ok := data.GetOk("webhook"); ok {
		webhook, diags := newMonitorV2WebhookDestinationInput(data, "webhook.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Webhook = webhook
	}
	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return input, diags
}

func newMonitorV2EmailDestinationInput(data *schema.ResourceData, path string) (email *gql.MonitorV2EmailDestinationInput, diags diag.Diagnostics) {
	// instantiation
	email = &gql.MonitorV2EmailDestinationInput{}

	// optionals
	if _, ok := data.GetOk(fmt.Sprintf("%susers", path)); ok {
		email.Users = make([]types.UserIdScalar, 0)
		for i := range data.Get(fmt.Sprintf("%susers", path)).([]interface{}) {
			uidStr := data.Get(fmt.Sprintf("%susers.%d", path, i)).(string)
			uid, err := types.StringToUserIdScalar(uidStr)
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

func newMonitorV2WebhookDestinationInput(data *schema.ResourceData, path string) (webhook *gql.MonitorV2WebhookDestinationInput, diags diag.Diagnostics) {
	// required
	url := data.Get(fmt.Sprintf("%surl", path)).(string)
	method := gql.MonitorV2HttpType(toCamel(data.Get(fmt.Sprintf("%smethod", path)).(string)))

	// instantiation
	webhook = &gql.MonitorV2WebhookDestinationInput{
		Url:    url,
		Method: method,
	}

	return webhook, diags
}
