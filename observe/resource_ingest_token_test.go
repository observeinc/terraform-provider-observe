package observe

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestIngestTokenEmpty(t *testing.T) {
	resource.ParallelTest(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{
				{
					Config: configPreamble + `
					resource "observe_ingest_token" "example" {
						workspace = data.observe_workspace.default.oid
					}
					`,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrPair("observe_ingest_token.example", "workspace", "data.observe_workspace.default", "oid"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "oid"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "name"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "description", ""),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "disabled", "false"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "secret"),
					),
				},
			},
		},
	)
}

func TestIngestTokenRegular(t *testing.T) {
	resource.ParallelTest(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{
				{
					Config: configPreamble + `
					resource "observe_ingest_token" "example" {
						workspace = data.observe_workspace.default.oid
						name = "name-0"
						description = "this is a description"
						disabled = false
					}
					`,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrPair("observe_ingest_token.example", "workspace", "data.observe_workspace.default", "oid"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "oid"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "name", "name-0"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "description", "this is a description"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "disabled", "false"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "secret"),
					),
				},
				{
					// Make sure we can disable a token and change its fields.
					Config: configPreamble + `
					resource "observe_ingest_token" "example" {
						workspace = data.observe_workspace.default.oid
						name = "name-1"
						description = "this is a new description"
						disabled = true
					}
					`,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrPair("observe_ingest_token.example", "workspace", "data.observe_workspace.default", "oid"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "oid"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "name", "name-1"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "description", "this is a new description"),
						resource.TestCheckResourceAttr("observe_ingest_token.example", "disabled", "true"),
						resource.TestCheckResourceAttrSet("observe_ingest_token.example", "secret"),
					),
				},
			},
		},
	)
}

func TestIngestTokenDisabled(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configPreamble + `
				resource "observe_ingest_token" "example" {
					workspace = data.observe_workspace.default.oid
					disabled = true
				}
				`,
				ExpectError: regexp.MustCompile("ingest token cannot be disabled on creation"),
			},
		},
	})
}
