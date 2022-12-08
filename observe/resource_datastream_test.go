package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	datastreamConfigPreamble = `
	resource "observe_datastream" "test" {
		workspace = data.observe_workspace.default.oid
		name      = "%[1]s"
	}`
)

func TestAccObserveDatastreamCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				  icon_url  = "test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_datastream.example", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_datastream.example", "dataset"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s-bis"
				  icon_url  = "changed"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix+"-bis"),
					resource.TestCheckResourceAttr("observe_datastream.example", "icon_url", "changed"),
					resource.TestCheckResourceAttrSet("observe_datastream.example", "dataset"),
				),
			},
		},
	})
}
