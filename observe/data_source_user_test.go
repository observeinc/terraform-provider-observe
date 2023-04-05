package observe

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_user" "system" {
				  email = "%s"
				}

				data "observe_user" "system_by_id" {
				  id = data.observe_user.system.id
				}

				`, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_user.system", "id"),
					resource.TestCheckResourceAttrSet("data.observe_user.system", "oid"),
					resource.TestCheckResourceAttr("data.observe_user.system", "email", systemUser()),
					resource.TestCheckResourceAttr("data.observe_user.system_by_id", "email", systemUser()),
				),
			},
		},
	})
}

func TestAccObserveSourceUserNotFound(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
                                data "observe_user" "default" {
                                  email = "%s"
                                }`, randomPrefix),
				ExpectError: regexp.MustCompile("user not found"),
			},
		},
	})
}

func systemUser() string {
	return fmt.Sprintf("system+%s@observeinc.com", os.Getenv("OBSERVE_CUSTOMER"))
}
