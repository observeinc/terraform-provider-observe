package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveHTTPPostCreate(t *testing.T) {
	t.Skip("OB-40979")
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data   = jsonencode({"hello"="%s"})
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_http_post.test", "acked"),
				),
			},
			{
				// no change when re-applying
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data   = jsonencode({"hello"="%s"})
				}`, randomPrefix),
				PlanOnly: true,
			},
			{
				// data change should force new instance
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data   = jsonencode({"data_change"="%s"})
				}`, randomPrefix),
			},
			{
				// tag change should force new instance
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data   = jsonencode({"hello"="%s"})
				  tags   = {"key" = "value"}
				}`, randomPrefix),
			},
			{
				// path change should force new instance
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  path   = "/hello"
				  data   = jsonencode({"hello"="%s"})
				}`, randomPrefix),
			},
		},
	})
}

func TestAccObserveHTTPPostCreateContentType(t *testing.T) {
	t.Skip("OB-40979")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "observe_http_post" "test" {
				  data         = "nota\n1-json"
				}`,
				ExpectError: regexp.MustCompile("failed to submit observations: bad request"),
			},
			{
				Config: `
				resource "observe_http_post" "test" {
				  data         = "a,b,c\n1,2,3\n4,5,6"
				  content_type = "text/csv"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_http_post.test", "acked"),
				),
			},
		},
	})
}
