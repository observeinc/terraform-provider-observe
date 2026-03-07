package observe

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccObserveServiceAccountToken(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%[1]s"
						description = "Test service account for token"
					}

					resource "observe_service_account_token" "test" {
						service_account = observe_service_account.test.oid
						label           = "%[1]s-token"
						description     = "Test API token"
						lifetime_hours  = 24
						disabled        = false
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_service_account_token.test", "service_account", "observe_service_account.test", "oid"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "label", randomPrefix+"-token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "description", "Test API token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "lifetime_hours", "24"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "disabled", "false"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "secret"),
					testCheckExpiration("observe_service_account_token.test", "expiration", time.Now().Add(24*time.Hour)),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%[1]s"
						description = "Test service account for token"
					}

					resource "observe_service_account_token" "test" {
						service_account = observe_service_account.test.oid
						label           = "%[1]s-token-updated"
						description     = "Updated test API token"
						lifetime_hours  = 365 * 24
						disabled        = true
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_service_account_token.test", "service_account", "observe_service_account.test", "oid"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "label", randomPrefix+"-token-updated"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "description", "Updated test API token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "lifetime_hours", "8760"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "disabled", "true"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "secret"),
					testCheckExpiration("observe_service_account_token.test", "expiration", time.Now().Add(365*24*time.Hour)),
				),
			},
		},
	})
}

func TestAccObserveServiceAccountTokenMinimal(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "minimal" {
						label = "%s"
					}

					resource "observe_service_account_token" "minimal" {
						service_account = observe_service_account.minimal.oid
						label           = "%s-token"
						lifetime_hours  = 1
					}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("observe_service_account_token.minimal", "service_account", "observe_service_account.minimal", "oid"),
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "label", randomPrefix+"-token"),
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "lifetime_hours", "1"),
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "disabled", "false"), // default value
					resource.TestCheckResourceAttrSet("observe_service_account_token.minimal", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.minimal", "secret"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.minimal", "expiration"),
				),
			},
		},
	})
}

func testCheckExpiration(resourceName string, fieldName string, expectedValue time.Time) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		attr, ok := rs.Primary.Attributes[fieldName]
		if !ok {
			return fmt.Errorf("attribute not found: %s", fieldName)
		}

		expiration, err := time.Parse(time.RFC3339, attr)
		if err != nil {
			return fmt.Errorf("could not parse expiration time: %w", err)
		}

		if expiration.Sub(expectedValue).Abs() > 5*time.Minute {
			return fmt.Errorf("expiration time is not within 5 minutes of expected value: %s", expiration)
		}

		return nil
	}
}
