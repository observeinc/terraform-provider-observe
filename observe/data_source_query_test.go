package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceQueryBadPipeline(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					data "observe_query" "%s" {
				      start = timestamp()

					  inputs = { "observation" = data.observe_dataset.observation.oid }

					  stage {
						pipeline = <<-EOF
						  error
						EOF
				      }
					}
				`, randomPrefix),
				ExpectError: regexp.MustCompile("unknown verb"),
			},
		},
	})
}

// TestAccObserveSourceQuery runs a query - we don't yet expect any data to be returned
func TestAccObserveSourceQuery(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
					data "observe_query" "test" {
				      start = timeadd(timestamp(), "-10m")

					  inputs = { "observation" = data.observe_dataset.observation.oid }

					  stage {
						pipeline = <<-EOF
						  filter true
						EOF
				      }
				  }
				`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_query.test", "id"),
				),
			},
		},
	})
}
