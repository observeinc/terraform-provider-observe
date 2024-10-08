package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveGrantGroupDatasetCreate(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
				  name      = "%[1]s"
				}

				resource "observe_grant" "example" {
				  subject = observe_rbac_group.example.oid
				  role    = "dataset_create"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_grant.example", "subject"),
					resource.TestCheckResourceAttr("observe_grant.example", "role", "dataset_create"),
				),
			},
		},
	})
}

func TestAccObserveGrantUserDatastreamEdit(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				data "observe_user" "system" {
				  email = "%[2]s"
				}

				resource "observe_grant" "example" {
				  subject = data.observe_user.system.oid
				  role    = "datastream_edit"
				  qualifier {
				    oid = observe_datastream.test.oid
				  }
				}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_grant.example", "subject"),
					resource.TestCheckResourceAttr("observe_grant.example", "role", "datastream_edit"),
					resource.TestCheckResourceAttr("observe_grant.example", "qualifier.#", "1"),
					resource.TestCheckResourceAttrSet("observe_grant.example", "qualifier.0.oid"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				data "observe_user" "system" {
				  email = "%[2]s"
				}

				resource "observe_grant" "example" {
				  subject = data.observe_user.system.oid
				  role    = "datastream_view"
				  qualifier {
				    oid = observe_datastream.test.oid
				  }
				}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_grant.example", "subject"),
				),
			},
		},
	})
}

func TestAccObserveGrantEveryoneWorksheetView(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				data "observe_rbac_group" "everyone" {
				  name = "everyone"
				}

				data "observe_oid" "dataset" {
				  oid = observe_datastream.test.dataset
				}

				resource "observe_worksheet" "example" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%[1]s"
				  queries   = <<-EOF
				  [{
					"id": "stage1",
					"input": [{
					  "inputName": "kubernetes/Container Logs",
					  "inputRole": "Data",
					  "datasetId": "${data.observe_oid.dataset.id}"
				    }]
				  }]
				  EOF
				}

				resource "observe_grant" "example" {
				  subject = data.observe_rbac_group.everyone.oid
				  role    = "worksheet_view"
				  qualifier {
				    oid = observe_worksheet.example.oid
				  }
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_grant.example", "subject"),
					resource.TestCheckResourceAttr("observe_grant.example", "role", "worksheet_view"),
					resource.TestCheckResourceAttr("observe_grant.example", "qualifier.#", "1"),
					resource.TestCheckResourceAttrSet("observe_grant.example", "qualifier.0.oid"),
				),
			},
		},
	})
}

func TestAccObserveGrantGroupAdminWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_rbac_group" "example" {
				  name      = "%[1]s"
				}

				resource "observe_grant" "example" {
				  subject = observe_rbac_group.example.oid
				  role    = "admin_workspace"
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_grant.example", "subject"),
					resource.TestCheckResourceAttr("observe_grant.example", "role", "admin_workspace"),
				),
			},
		},
	})
}
