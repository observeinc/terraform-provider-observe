package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// ============================================================================
// Inbound Share Data Source Tests
// ============================================================================
//
// These tests verify the observe_inbound_share data source which looks up
// existing external shares that have been imported into Observe.
//
// The data source supports two lookup methods:
//   1. By share_name + provider_account (tested here)
//   2. By share ID directly
//
// Test Prerequisites:
//   - Share MATTG_SHAREIN_TEST_DATA_SHARE2 must exist
//   - Share must be from provider HC83707.OBSERVE_O2_1
//   - Share must be in Active or Healthy status
//
// ============================================================================

// TestAccObserveInboundShareDataSource_LookupByName tests share lookup by name and provider.
//
// This test verifies:
//   - Data source can find shares by Snowflake share name + provider account
//   - Share metadata (ID, OID, status, provider type) is correctly populated
//   - The snowflake_share_name field returns the actual Snowflake share name
//   - Provider type is correctly identified as "Snowflake"
//
// Lookup Method:
//
//	The data source uses the share's Snowflake configuration (share name + provider account)
//	to uniquely identify the share. This is more reliable than using the Observe display name
//	which may not be unique.
//
// What this validates:
//   - REST API client's LookupShare function works correctly
//   - Share exists and is accessible via the API
//   - All share metadata fields are properly mapped to Terraform attributes
func TestAccObserveInboundShareDataSource_LookupByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Configure data source to look up share by snowflake_config
				Config: fmt.Sprintf(`
					# Look up an inbound share by its Snowflake share name and provider account
					data "observe_inbound_share" "test" {
						snowflake_config {
							share_name       = "%s"  # Snowflake share name
							provider_account = "%s"  # Snowflake provider account (format: ORG.ACCOUNT)
						}
					}
				`, testInboundShareName, testInboundShareProvider),
				Check: resource.ComposeTestCheckFunc(
					// Verify share ID fields are populated
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "id"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "oid"),

					// Verify snowflake_config output matches what we looked up
					resource.TestCheckResourceAttr("data.observe_inbound_share.test", "snowflake_config.0.share_name", testInboundShareName),
					resource.TestCheckResourceAttr("data.observe_inbound_share.test", "snowflake_config.0.provider_account", testInboundShareProvider),
					resource.TestCheckResourceAttr("data.observe_inbound_share.test", "provider_type", "Snowflake"),

					// Verify share is in a valid operational state
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "status"),

					// Additional metadata should be present
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share.test", "updated_at"),
				),
			},
		},
	})
}
