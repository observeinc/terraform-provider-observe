package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveServiceAccountDataSource(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_service_account" "test" {
						label       = "%s"
						description = "Test service account for data source"
						disabled    = false
					}

					data "observe_service_account" "by_id" {
						id = observe_service_account.test.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					// Check data source by ID
					resource.TestCheckResourceAttr("data.observe_service_account.by_id", "label", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_service_account.by_id", "description", "Test service account for data source"),
					resource.TestCheckResourceAttr("data.observe_service_account.by_id", "disabled", "false"),

					// Verify the data source returns the same service account as the resource
					resource.TestCheckResourceAttrPair("observe_service_account.test", "id", "data.observe_service_account.by_id", "id"),
					resource.TestCheckResourceAttrPair("observe_service_account.test", "oid", "data.observe_service_account.by_id", "oid"),
				),
			},
		},
	})
}
