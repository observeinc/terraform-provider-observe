package observe

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

var filedropConfigPreamble = configPreamble + datastreamConfigPreamble

func TestAccObserveFiledrop(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	filedropRoleArn := os.Getenv("OBSERVE_FILEDROP_ROLE_ARN")
	if os.Getenv("CI") != "true" {
		// The role_arn `OBSERVE_FILEDROP_ROLE_ARN` was manually created for the provider CI Observe account.
		// This test fails if the role_arn does not exist
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's AWS account.")
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(filedropConfigPreamble+`
				resource "observe_filedrop" "example" {
					workspace  = data.observe_workspace.default.oid
					datastream = observe_datastream.test.oid
					config {
						provider {
							aws {
								region  = "us-west-2"
								role_arn = "%[2]s"
							}
						}
					}
				}`, randomPrefix, filedropRoleArn),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "name"),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "status"),
					resource.TestCheckResourceAttr("observe_filedrop.example", "config.0.provider.0.aws.0.region", "us-west-2"),
					resource.TestCheckResourceAttr("observe_filedrop.example", "config.0.provider.0.aws.0.role_arn", filedropRoleArn),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.arn"),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.bucket"),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.prefix"),
				),
			},
			{
				Config: fmt.Sprintf(filedropConfigPreamble+`
				resource "observe_filedrop" "example" {
					workspace  = data.observe_workspace.default.oid
					name       = "%[1]s"
					datastream = observe_datastream.test.oid
					config {
						provider {
							aws {
								region  = "us-west-2"
								role_arn = "%[2]s"
							}
						}
					}
				}`, randomPrefix, filedropRoleArn),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_filedrop.example", "name", randomPrefix),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "status"),
					resource.TestCheckResourceAttr("observe_filedrop.example", "config.0.provider.0.aws.0.region", "us-west-2"),
					resource.TestCheckResourceAttr("observe_filedrop.example", "config.0.provider.0.aws.0.role_arn", filedropRoleArn),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.arn"),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.bucket"),
					resource.TestCheckResourceAttrSet("observe_filedrop.example", "endpoint.0.s3.0.prefix"),
				),
			},
		},
	})
}
