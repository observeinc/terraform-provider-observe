package observe

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveServiceAccountToken(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	now := time.Now().UTC()
	initialExpiration := now.Add(720 * time.Hour).Format(time.RFC3339)
	updatedExpiration := now.Add(168 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%[1]s"
						description = "Test service account for API token"
						disabled    = false
					}

					resource "observe_service_account_token" "test" {
						service_account = observe_service_account.test.oid
						label           = "%[1]s-token"
						description     = "Test API token"
						expiration      = "%[2]s"
					}
				`, randomPrefix, initialExpiration),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account_token.test", "label", randomPrefix+"-token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "description", "Test API token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "disabled", "false"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "expiration", initialExpiration),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "secret"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%[1]s"
						description = "Test service account for API token"
						disabled    = false
					}

					resource "observe_service_account_token" "test" {
						service_account = observe_service_account.test.oid
						label           = "%[1]s-token-updated"
						description     = "Updated test API token"
						expiration      = "%[2]s"
						disabled        = true
					}
				`, randomPrefix, updatedExpiration),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account_token.test", "label", randomPrefix+"-token-updated"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "description", "Updated test API token"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_service_account_token.test", "expiration", updatedExpiration),
					resource.TestCheckResourceAttrSet("observe_service_account_token.test", "secret"),
				),
			},
		},
	})
}

func TestAccObserveServiceAccountTokenMinimal(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	expiration := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "minimal" {
						label = "%[1]s"
					}

					resource "observe_service_account_token" "minimal" {
						service_account = observe_service_account.minimal.oid
						label           = "%[1]s-token"
						expiration      = "%[2]s"
					}
				`, randomPrefix, expiration),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "label", randomPrefix+"-token"),
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "disabled", "false"), // default value
					resource.TestCheckResourceAttr("observe_service_account_token.minimal", "expiration", expiration),
					resource.TestCheckResourceAttrSet("observe_service_account_token.minimal", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account_token.minimal", "secret"),
				),
			},
		},
	})
}
