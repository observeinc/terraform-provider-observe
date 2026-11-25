package observe

import (
	"fmt"
	"regexp"
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
					// The IdP owns the label after creation, so the rename is ignored.
					resource.TestCheckResourceAttr("observe_user.example", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_user.example", "comment", "managed by terraform"),
					resource.TestCheckResourceAttr("observe_user.example", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_user.example", "status", "UserStatusDisabled"),
				),
			},
			{
				// Observe cannot update SSO identities, so this fails at plan.
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_user" "example" {
					email             = "%[1]s"
					label             = "%[2]s"
					identity_provider = "GOOGLE"
					identity          = "%[2]s-google-sub"
					comment           = "managed by terraform"
					disabled          = true
				}
				`, email, randomPrefix),
				ExpectError: regexp.MustCompile(`identity_provider cannot be changed`),
			},
			{
				ResourceName:      "observe_user.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccObserveUserExistingEmail checks that Create refuses an email that
// already belongs to a user instead of adopting it or creating a duplicate.
func TestAccObserveUserExistingEmail(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	email := randomPrefix + "@example.com"
	user := func(name string) string {
		return fmt.Sprintf(`
		resource "observe_user" "%[3]s" {
			email             = "%[1]s"
			label             = "%[2]s"
			identity_provider = "SAML2"
			identity          = "%[1]s"
			disabled          = true
		}
		`, email, randomPrefix, name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configPreamble + user("first"),
				Check:  resource.TestCheckResourceAttr("observe_user.first", "status", "UserStatusDisabled"),
			},
			{
				// The first user is disabled, which is also what a destroyed
				// user looks like; it must still be refused.
				Config:      configPreamble + user("first") + user("second"),
				ExpectError: regexp.MustCompile(`already exists`),
			},
		},
	})
}
