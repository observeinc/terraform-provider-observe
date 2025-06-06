package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveCloudInfo(t *testing.T) {
	t.Skip()
	t.Skip()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				data "observe_cloud_info" "current" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_cloud_info.current", "cloud_provider"),
					resource.TestCheckResourceAttrSet("data.observe_cloud_info.current", "account_id"),
					resource.TestCheckResourceAttrSet("data.observe_cloud_info.current", "region"),
				),
			},
		},
	})
}
