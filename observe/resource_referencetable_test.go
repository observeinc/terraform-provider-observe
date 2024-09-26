package observe

import (
    "fmt"
    "testing"

    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveReferenceTable(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
    
    resource.Test(t, resource.TestCase{
        PreCheck: func() { testAccPreCheck(t) },
        Providers: testAccProviders,
        Steps: []resource.TestStep{
            {
                Config: fmt.Sprintf(configPreamble + `
                resource "observe_referencetable" "example" {
                    workspace = data.observe_workspace.default.oid
                    name = "%s"
                    icon_url = "test"
                }
                `, randomPrefix),
                Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_referencetable.example", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_referencetable.example", "icon_url", "test"),
					resource.TestCheckResourceAttrSet("observe_referencetable.example", "dataset"),
				),

            },
        },
    })
}
