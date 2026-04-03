package observe

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccProtoV5ProviderFactories map[string]func() (tfprotov5.ProviderServer, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"observe": testAccProvider,
	}
	testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
		"observe": func() (tfprotov5.ProviderServer, error) {
			ctx := context.Background()
			providers := []func() tfprotov5.ProviderServer{
				Provider().GRPCProvider,
				providerserver.NewProtocol5(NewFrameworkProvider()),
			}
			muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
			if err != nil {
				return nil, err
			}
			return muxServer.ProviderServer(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	requiredEnvVars := []string{"OBSERVE_CUSTOMER", "OBSERVE_DOMAIN"}

	for _, k := range requiredEnvVars {
		if v := os.Getenv(k); v == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}
