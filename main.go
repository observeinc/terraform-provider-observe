package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/observeinc/terraform-provider-observe/observe"
)

//go:generate env -u OBSERVE_CUSTOMER go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
// `env -u` unsets a variable.  This prevents the go generate command from inheriting it.
// When set, `OBSERVE_CUSTOMER` defines the default for the `customer` provider attribute.
// This makes this required field optional, since a default is set.

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: observe.Provider})
}
