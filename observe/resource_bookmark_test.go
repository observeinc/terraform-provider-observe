package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	// common to all configs
	bookmarkConfigPreamble = configPreamble + datastreamConfigPreamble + `
		resource "observe_bookmark_group" "a" {
			workspace 	 = data.observe_workspace.default.oid
			name      	 = "%[1]s-a"
		}

		resource "observe_bookmark_group" "b" {
			workspace 	 = data.observe_workspace.default.oid
			name      	 = "%[1]s-b"
		}
		`
	bookmarkDashboardConfigPreamble = bookmarkConfigPreamble + dashboardConfigPreamble
)

func TestAccObserveBookmarkCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group  = observe_bookmark_group.a.oid
				  target = observe_datastream.test.dataset
				  name   = "Test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "name", "Test"),
				),
			},
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group    = observe_bookmark_group.a.oid
				  target   = observe_datastream.test.dataset
				  name     = "Test"
				  icon_url = "star"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "icon_url", "star"),
				),
			},
		},
	})
}

func TestAccObserveBookmarkMoveGroup(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group  = observe_bookmark_group.a.oid
				  target = observe_datastream.test.dataset
				  name   = "Test"
				}
				`, randomPrefix),
			},
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group  = observe_bookmark_group.b.oid
				  target = observe_datastream.test.dataset
				  name   = "Test"
				}
				`, randomPrefix),
			},
		},
	})
}

func TestAccObserveBookmarkDashboard(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(bookmarkDashboardConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group  = observe_bookmark_group.a.oid
				  target = observe_dashboard.first.oid
				  name   = "Test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "name", "Test"),
					resource.TestCheckResourceAttrPair("observe_bookmark.bm", "target", "observe_dashboard.first", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(bookmarkDashboardConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group    = observe_bookmark_group.a.oid
				  target   = observe_dashboard.first.oid
				  name     = "Test"
				  icon_url = "star"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "icon_url", "star"),
					resource.TestCheckResourceAttrPair("observe_bookmark.bm", "target", "observe_dashboard.first", "oid"),
				),
			},
		},
	})
}

func TestAccObserveBookmarkKind(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group  = observe_bookmark_group.a.oid
				  target = observe_datastream.test.dataset
				  name   = "Test"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "name", "Test"),
					resource.TestCheckResourceAttr("observe_bookmark.bm", "bookmark_kind", ""),
				),
			},
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group         = observe_bookmark_group.a.oid
				  target        = observe_datastream.test.dataset
				  name          = "Test"
				  bookmark_kind = "log_explorer"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "name", "Test"),
					resource.TestCheckResourceAttr("observe_bookmark.bm", "bookmark_kind", "log_explorer"),
				),
			},
			{
				Config: fmt.Sprintf(bookmarkConfigPreamble+`
				resource "observe_bookmark" "bm" {
				  group         = observe_bookmark_group.a.oid
				  target        = observe_datastream.test.dataset
				  name          = "Test"
				  bookmark_kind = "metric_explorer"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_bookmark.bm", "name", "Test"),
					resource.TestCheckResourceAttr("observe_bookmark.bm", "bookmark_kind", "metric_explorer"),
				),
			},
		},
	})
}
