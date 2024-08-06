package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

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
