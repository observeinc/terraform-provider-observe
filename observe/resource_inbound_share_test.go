package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testInboundShareRestingLabel is the label the shared test share is left with.
const testInboundShareRestingLabel = "tf-acc inbound share"

// TestAccObserveInboundShare_ImportAndUpdate manages the registration of the
// shared test share without ever creating or destroying it. The table tests
// depend on that registration, and destroying it would unmount the share for
// every run until someone registers it again. Create is not covered: it
// rejects a share that is already registered.
//
// Test flow:
//
//	Step 1: Look up the share ID with the data source
//	Step 2: Import the registration into state
//	Step 3: Change the label
//	Step 4: Set the label back to testInboundShareRestingLabel
//	Step 5: Drop the resource from state with a removed block, so the final
//	        destroy leaves the registration in place
func TestAccObserveInboundShare_ImportAndUpdate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundShareLookupConfig(),
			},
			{
				Config:       testAccInboundShareResourceConfig(testInboundShareRestingLabel),
				ResourceName: "observe_inbound_share.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["data.observe_inbound_share.test"]
					if !ok {
						return "", fmt.Errorf("data.observe_inbound_share.test not found in state")
					}
					return rs.Primary.ID, nil
				},
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported share, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["provider_account"] != testInboundShareProvider {
						return fmt.Errorf("provider_account = %q, want %q", attrs["provider_account"], testInboundShareProvider)
					}
					if attrs["oid"] == "" || attrs["status"] == "" {
						return fmt.Errorf("oid and status must be set after import, got %q and %q", attrs["oid"], attrs["status"])
					}
					return nil
				},
			},
			{
				Config: testAccInboundShareResourceConfig(randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_inbound_share.test", "label", randomPrefix),
					resource.TestCheckResourceAttrPair("observe_inbound_share.test", "id", "data.observe_inbound_share.test", "id"),
					resource.TestCheckResourceAttr("observe_inbound_share.test", "provider_account", testInboundShareProvider),
				),
			},
			{
				Config: testAccInboundShareResourceConfig(testInboundShareRestingLabel),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_inbound_share.test", "label", testInboundShareRestingLabel),
				),
			},
			{
				Config: testAccInboundShareLookupConfig() + `
removed {
	from = observe_inbound_share.test

	lifecycle {
		destroy = false
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "id"),
				),
			},
		},
	})
}

func testAccInboundShareLookupConfig() string {
	return fmt.Sprintf(`
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}
`, testInboundShareName, testInboundShareProvider)
}

func testAccInboundShareResourceConfig(label string) string {
	return testAccInboundShareLookupConfig() + fmt.Sprintf(`
resource "observe_inbound_share" "test" {
	share_name       = "%s"
	provider_account = "%s"
	label            = "%s"

	# If a step fails, the post-test destroy must not unregister the shared share.
	lifecycle {
		prevent_destroy = true
	}
}
`, testInboundShareName, testInboundShareProvider, label)
}
