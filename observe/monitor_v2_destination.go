package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

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

func newMonitorV2DestinationInput(actInput *gql.MonitorV2ActionInput) (input *gql.MonitorV2DestinationInput, diags diag.Diagnostics) {
	// required
	name := "my destination" // name is ignored for inlined destinations, so we can set it to any value.

	// instantiation
	inlineVal := true // we are currently only allowing destinations to be inlined
	input = &gql.MonitorV2DestinationInput{
		Type:   actInput.Type,
		Name:   name,
		Inline: &inlineVal,
	}

	if actInput.Email != nil {
		input.Email.Users = make([]types.UserIdScalar, 0)
		for _, usr := range actInput.Email.Users {
			input.Email.Users = append(input.Email.Users, usr)
		}
		input.Email.Addresses = make([]string, 0)
		for _, addr := range actInput.Email.Addresses {
			input.Email.Addresses = append(input.Email.Addresses, addr)
		}
	}

	if actInput.Webhook != nil {
		input.Webhook.Method = *actInput.Webhook.Method
		input.Webhook.Url = *actInput.Webhook.Url
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
