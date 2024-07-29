package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatastreamTokenCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_workspace" "example" {
				  name      = "%[1]s"
				}

				resource "observe_datastream" "example" {
				  workspace = observe_workspace.example.oid
				  name      = "Hello"
				}

				resource "observe_datastream_token" "example" {
				  datastream = observe_datastream.example.oid
				  name      = "World"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream_token.example", "name", "World"),
					resource.TestCheckResourceAttrSet("observe_datastream_token.example", "secret"),
					resource.TestCheckResourceAttrPair("observe_datastream_token.example", "datastream", "observe_datastream.example", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "observe_workspace" "example" {
				  name      = "%[1]s"
				}

				resource "observe_datastream" "example" {
				  workspace = observe_workspace.example.oid
				  name      = "Hello"
				}

				resource "observe_datastream_token" "example" {
				  datastream = observe_datastream.example.oid
				  name      = "Worlds"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_datastream_token.example", "name", "Worlds"),
					resource.TestCheckResourceAttrSet("observe_datastream_token.example", "secret"),
					resource.TestCheckResourceAttrPair("observe_datastream_token.example", "datastream", "observe_datastream.example", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "observe_workspace" "example" {
				  name      = "%[1]s"
				}

				resource "observe_datastream" "example" {
				  workspace = observe_workspace.example.oid
				  name      = "Hello"
				}

				resource "observe_datastream_token" "example" {
				  datastream = observe_datastream.example.oid
				  name      = "SecretWorlds"
				  password	= "Very-Very-Secret-Long-Hidden-Password"
				}
				`, randomPrefix),
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
