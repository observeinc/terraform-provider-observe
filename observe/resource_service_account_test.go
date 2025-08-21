package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveServiceAccount(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%s"
						description = "Test service account"
						disabled    = false
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account.test", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_service_account.test", "description", "Test service account"),
					resource.TestCheckResourceAttr("observe_service_account.test", "disabled", "false"),
					resource.TestCheckResourceAttrSet("observe_service_account.test", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account.test", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%s-updated"
						description = "Updated Test service account"
						disabled    = true
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account.test", "label", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_service_account.test", "description", "Updated Test service account"),
					resource.TestCheckResourceAttr("observe_service_account.test", "disabled", "true"),
				),
			},
		},
	})
}

func TestAccObserveServiceAccountMinimal(t *testing.T) {
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
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_service_account.minimal", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_service_account.minimal", "disabled", "false"), // default value
					resource.TestCheckResourceAttrSet("observe_service_account.minimal", "id"),
					resource.TestCheckResourceAttrSet("observe_service_account.minimal", "oid"),
				),
			},
		},
	})
}
