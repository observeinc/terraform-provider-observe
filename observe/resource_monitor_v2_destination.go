package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			// "name": { // String! 			for inline actions, name is ignored.
			// 	Type:     schema.TypeString,
			// 	Required: true,
			// },
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

func monitorV2FlattenDestination(gqlDest gql.MonitorV2Destination) []interface{} {
	dest := map[string]interface{}{
		"oid": gqlDest.Oid().String(),
	}

	// optional
	if gqlDest.Description != nil {
		dest["description"] = *gqlDest.Description
	}

	if gqlDest.Email != nil {
		dest["email"] = monitorV2FlattenEmailDestination(*gqlDest.Email)
	}

	if gqlDest.IconUrl != nil {
		dest["icon_url"] = *gqlDest.IconUrl
	}

	if gqlDest.Webhook != nil {
		dest["webhook"] = monitorv2FlattenWebhookDestination(*gqlDest.Webhook)
	}

	return []interface{}{dest}
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
		userOIDStrs := make([]string, 0)
		for _, uid := range gqlEmail.Users {
			userOID := oid.UserOid(uid)
			userOIDStrs = append(userOIDStrs, userOID.String())
		}
		email["users"] = userOIDStrs
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

func newMonitorV2DestinationInput(data *schema.ResourceData, path string, actionType gql.MonitorV2ActionType) (input *gql.MonitorV2DestinationInput, diags diag.Diagnostics) {
	// required
	name := "my destination" // name is ignored for inlined destinations, so we can set it to any value.

	// instantiation
	inlineVal := true // we are currently only allowing destinations to be inlined
	input = &gql.MonitorV2DestinationInput{
		Type:   actionType,
		Name:   name,
		Inline: &inlineVal,
	}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%sdescription", path)); ok {
		input.Description = stringPtr(v.(string))
	}
	if _, ok := data.GetOk(fmt.Sprintf("%semail", path)); ok {
		email, diags := newMonitorV2EmailDestinationInput(data, fmt.Sprintf("%semail.0.", path))
		if diags.HasError() {
			return nil, diags
		}
		input.Email = email
	}
	if _, ok := data.GetOk(fmt.Sprintf("%swebhook", path)); ok {
		webhook, diags := newMonitorV2WebhookDestinationInput(data, fmt.Sprintf("%swebhook.0.", path))
		if diags.HasError() {
			return nil, diags
		}
		input.Webhook = webhook
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sicon_url", path)); ok {
		input.IconUrl = stringPtr(v.(string))
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sdescription", path)); ok {
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
