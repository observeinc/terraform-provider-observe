package observe

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveHTTPPostCreate(t *testing.T) {
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

// TestAccObserveHTTPPostRefresh verifies that once we exceed a given
// refresh duration, our observation resource is "recomputed": destroyed and
// then recreated.
func TestAccObserveHTTPPostRefresh(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	// we want to specify the lowest possible refresh time, but we need to
	// allow test framework to run plan after apply in order to verify diff is
	// clean.
	minRefresh := 3 * time.Second

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data    = jsonencode({"hello"="%s"})
				  refresh = "%s"
				}`, randomPrefix, minRefresh),
			},
			{
				// after sleeping minRefresh, we expect resource to be
				// recomputed
				PreConfig: func() {
					time.Sleep(minRefresh)
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: fmt.Sprintf(`
				resource "observe_http_post" "test" {
				  data   = jsonencode({"hello"="%s"})
				  refresh = "%s"
				}`, randomPrefix, minRefresh),
			},
		},
	})
}
