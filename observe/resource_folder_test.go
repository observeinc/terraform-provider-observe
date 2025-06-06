package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveFolderCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_folder" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s"
					icon_url  = "test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_folder.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_folder.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_folder.example", "description", ""),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_folder" "example" {
					workspace    = data.observe_workspace.default.oid
					name         = "%[1]s-1"
					icon_url     = "test"
					description  = "a description"
				}
				`, randomPrefix, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_folder.example", "name", randomPrefix+"-1"),
					resource.TestCheckResourceAttr("observe_folder.example", "icon_url", "test"),
					resource.TestCheckResourceAttr("observe_folder.example", "description", "a description"),
				),
			},
		},
	})
}
