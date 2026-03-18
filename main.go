package main

import (
	"context"
	"log"
	_ "time/tzdata" // Embed timezone database for environments without system tzdata

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/observeinc/terraform-provider-observe/observe"
)

//go:generate env -u OBSERVE_CUSTOMER go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
// `env -u` unsets a variable.  This prevents the go generate command from inheriting it.
// When set, `OBSERVE_CUSTOMER` defines the default for the `customer` provider attribute.
// This makes this required field optional, since a default is set.

func main() {
	ctx := context.Background()

	providers := []func() tfprotov5.ProviderServer{
		observe.Provider().GRPCProvider,
		providerserver.NewProtocol5(observe.NewFrameworkProvider()),
	}

	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	err = tf5server.Serve(
		"registry.terraform.io/observeinc/observe",
		muxServer.ProviderServer,
	)
	if err != nil {
		log.Fatal(err)
	}
}
