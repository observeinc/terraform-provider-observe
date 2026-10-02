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
// replace the datastream, which would delete its dataset and data, unless
// force_destroy opts into the replacement.
func TestAccObserveDatastreamRejectsTypeChange(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := func(extra string) string {
		return fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					%s
				}
				`, randomPrefix, extra)
	}
	var originalID string

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// An Any datastream leaves type unset in state.
				Config: config(""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("observe_datastream.example", "type"),
					resource.TestCheckResourceAttrWith("observe_datastream.example", "id", func(id string) error {
						originalID = id
						return nil
					}),
				),
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
			{
				// Setting force_destroy alone keeps the datastream.
				Config: config(`force_destroy = true`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "force_destroy", "true"),
					resource.TestCheckResourceAttrWith("observe_datastream.example", "id", func(id string) error {
						if id != originalID {
							return fmt.Errorf("id = %s, want unchanged %s", id, originalID)
						}
						return nil
					}),
				),
			},
			{
				// With force_destroy, changing type replaces the datastream.
				Config: config(`
					type          = "OtelLogs"
					force_destroy = true`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream.example", "type", "OtelLogs"),
					resource.TestCheckResourceAttrWith("observe_datastream.example", "id", func(id string) error {
						if id == originalID {
							return fmt.Errorf("id = %s, want a replacement datastream", id)
						}
						return nil
					}),
				),
			},
		},
	})
}
