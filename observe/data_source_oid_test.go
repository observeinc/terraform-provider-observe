package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDataOID_Parse(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configPreamble + `
					data "observe_oid" "workspace" {
						oid = data.observe_workspace.default.oid
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_oid.workspace", "type", "workspace"),
					resource.TestCheckResourceAttrPair("data.observe_oid.workspace", "id", "data.observe_workspace.default", "id"),
				),
			},
		},
	})
}

func TestAccObserveDataOID_Format(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configPreamble + `
					data "observe_oid" "workspace" {
						id = data.observe_workspace.default.id
						type = "workspace"
					}
			`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.observe_oid.workspace", "oid", "data.observe_workspace.default", "oid"),
				),
			},
		},
	})
}
