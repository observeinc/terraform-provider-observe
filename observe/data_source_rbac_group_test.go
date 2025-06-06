package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	defaultRbacGroupReaderName = "reader"
)

func TestAccObserveSourceRbacGroup(t *testing.T) {
	t.Skip()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "observe_rbac_group" "reader" {
					name = "%s"
				}

				data "observe_rbac_group" "reader_by_id" {
					id = data.observe_rbac_group.reader.id
				}

				`, defaultRbacGroupReaderName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_rbac_group.reader", "id"),
					resource.TestCheckResourceAttrSet("data.observe_rbac_group.reader", "oid"),
					resource.TestCheckResourceAttr("data.observe_rbac_group.reader", "name", defaultRbacGroupReaderName),
					resource.TestCheckResourceAttr("data.observe_rbac_group.reader_by_id", "name", defaultRbacGroupReaderName),
				),
			},
		},
	})
}
