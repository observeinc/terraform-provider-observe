package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveUserCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	email := randomPrefix + "@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_user" "example" {
					email             = "%[1]s"
					label             = "%[2]s"
					identity_provider = "SAML2"
					identity          = "%[1]s"
				}
				`, email, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_user.example", "email", email),
					resource.TestCheckResourceAttr("observe_user.example", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_user.example", "identity_provider", "SAML2"),
					resource.TestCheckResourceAttr("observe_user.example", "identity", email),
					resource.TestCheckResourceAttr("observe_user.example", "role", "reader"),
					resource.TestCheckResourceAttr("observe_user.example", "disabled", "false"),
					resource.TestCheckResourceAttr("observe_user.example", "status", "UserStatusActive"),
					resource.TestCheckResourceAttrSet("observe_user.example", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_user" "example" {
					email             = "%[1]s"
					label             = "%[2]s-renamed"
					identity_provider = "SAML2"
					identity          = "%[1]s"
					comment           = "managed by terraform"
					disabled          = true
				}
				`, email, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_user.example", "label", randomPrefix+"-renamed"),
					resource.TestCheckResourceAttr("observe_user.example", "comment", "managed by terraform"),
					resource.TestCheckResourceAttr("observe_user.example", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_user.example", "status", "UserStatusDisabled"),
				),
			},
			{
				ResourceName:      "observe_user.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
