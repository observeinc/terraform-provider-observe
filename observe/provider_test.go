package observe

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"observe": testAccProvider,
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

// testAccPreCheckInboundShare verifies prerequisites for inbound share tests.
// Skips tests unless running in CI environment.
func testAccPreCheckInboundShare(t *testing.T) {
	// Run standard prechecks first
	testAccPreCheck(t)

	// Skip unless running in CI - these tests require external Snowflake share setup
	if os.Getenv("CI") != "true" {
		t.Skip("CI != true. Inbound share tests require manual Snowflake share setup that is only available in CI.")
	}

	// Note: TEST_INBOUND_* environment variables have defaults and don't need to be checked
}
