package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveRbacGroupCreate(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_group.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_rbac_group.example", "description", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
					name         = "%[1]s-1"
					description  = "a description"
				}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_rbac_group.example", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_rbac_group.example", "description", "a description"),
					resource.TestCheckResourceAttrSet("observe_rbac_group.example", "id"),
					resource.TestCheckResourceAttrSet("observe_rbac_group.example", "oid"),
				),
			},
		},
	})
}
