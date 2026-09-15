package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatastreamCreateOtelLogs(t *testing.T) {
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
					type      = "OtelLogs"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_datastream.example", "type", "OtelLogs"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace   = data.observe_workspace.default.oid
					name        = "%s-updated"
					description = "updated description"
					type        = "OtelLogs"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "name", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_datastream.example", "description", "updated description"),
					resource.TestCheckResourceAttr("observe_datastream.example", "type", "OtelLogs"),
				),
			},
			{
				ResourceName:      "observe_datastream.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
