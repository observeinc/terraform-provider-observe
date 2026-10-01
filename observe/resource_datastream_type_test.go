package observe

import (
	"fmt"
	"regexp"
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

// Setting type on an existing Any datastream must fail the plan rather than
// replace the datastream, which would delete its dataset and data.
func TestAccObserveDatastreamRejectsTypeChange(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := func(typeLine string) string {
		return fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					%s
				}
				`, randomPrefix, typeLine)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// An Any datastream leaves type unset in state.
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr("observe_datastream.example", "type"),
			},
			{
				Config:      config(`type = "OtelLogs"`),
				ExpectError: regexp.MustCompile(`type\s+cannot\s+be\s+changed\s+on\s+an\s+existing\s+datastream`),
			},
			{
				// The rejected plan left the Any datastream in place.
				Config:   config(""),
				PlanOnly: true,
			},
		},
	})
}
