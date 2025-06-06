package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatastreamTokenCreate(t *testing.T) {
	t.Skip()
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-hello"
				}

				resource "observe_datastream_token" "example" {
					datastream = observe_datastream.example.oid
					name      = "%[1]s-world"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream_token.example", "name", randomPrefix+"-world"),
					resource.TestCheckResourceAttrSet("observe_datastream_token.example", "secret"),
					resource.TestCheckResourceAttrPair("observe_datastream_token.example", "datastream", "observe_datastream.example", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-hello"
				}

				resource "observe_datastream_token" "example" {
					datastream = observe_datastream.example.oid
					name      = "%[1]s-worlds"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream_token.example", "name", randomPrefix+"-worlds"),
					resource.TestCheckResourceAttrSet("observe_datastream_token.example", "secret"),
					resource.TestCheckResourceAttrPair("observe_datastream_token.example", "datastream", "observe_datastream.example", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble + `
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "Hello"
				}

				resource "observe_datastream_token" "example" {
					datastream = observe_datastream.example.oid
					name      = "SecretWorlds"
					password	= "Very-Very-Secret-Long-Hidden-Password"
				}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream_token.example", "name", "SecretWorlds"),
					resource.TestCheckResourceAttr("observe_datastream_token.example", "id", "ds22hZTuuQwkqtbqWOGkSs2agrBwP0"),
					resource.TestCheckResourceAttr("observe_datastream_token.example", "secret", "ds22hZTuuQwkqtbqWOGkSs2agrBwP0:Very-Very-Secret-Long-Hidden-Password"),
					resource.TestCheckResourceAttrPair("observe_datastream_token.example", "datastream", "observe_datastream.example", "oid"),
				),
			},
		},
	})
}
