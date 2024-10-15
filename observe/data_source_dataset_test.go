package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveSourceDataset(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_datastream" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s"
					}

					data "observe_dataset" "a" {
						workspace = data.observe_workspace.default.oid
						name      = observe_datastream.a.name
					}
			`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dataset.a", "name", randomPrefix),
				),
			},
		},
	})
}

func TestAccObserveSourceDatasetStage(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_datastream" "a" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s-a"
					}

					resource "observe_dataset" "b" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s-b"

						inputs = { "a" = observe_datastream.a.dataset }

						stage {
							pipeline = <<-EOF
								filter false
							EOF
						}
					}

					data "observe_dataset" "lookup_by_name" {
						workspace  = data.observe_workspace.default.oid
						name       = observe_dataset.b.name
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_name", "name", randomPrefix+"-b"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_name", "stage.0.pipeline", "filter false\n"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
						resource "observe_datastream" "a" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s"
						}

						resource "observe_dataset" "b" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s-b"

							inputs = { "a" = observe_datastream.a.dataset }

							stage {
								pipeline = <<-EOF
									filter false
								EOF
							}
						}

						data "observe_dataset" "lookup_by_id" {
							id = observe_dataset.b.id
						}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dataset.lookup_by_id", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "name", randomPrefix+"-b"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "stage.0.pipeline", "filter false\n"),
				),
			},
		},
	})
}

func TestAccObserveSourceDatasetNotFound(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				data "observe_dataset" "missing" {
				  workspace = data.observe_workspace.default.oid
				  name      = "%s"
				}`, randomPrefix),
				ExpectError: regexp.MustCompile(randomPrefix),
			},
		},
	})
}

func TestAccObserveSourceDatasetEmptyString(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_dataset" "missing" {
				  workspace = data.observe_workspace.default.oid
				  name      = ""
				}`),
				ExpectError: regexp.MustCompile("not be an empty string"),
			},
		},
	})
}

func TestAccObserveSourceDatasetInvalidID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_dataset" "missing" {
				  workspace = data.observe_workspace.default.oid
				  id        = "xyz"
				}`),
				ExpectError: regexp.MustCompile("expected ID to be valid integer"),
			},
		},
	})
}

func TestAccObserveSourceDatasetStageCorrelationTag(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
						resource "observe_datastream" "a" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s"
						}

						resource "observe_dataset" "b" {
							workspace = data.observe_workspace.default.oid
							name      = "%[1]s-b"

							inputs = { "a" = observe_datastream.a.dataset }

							stage {
								pipeline = <<-EOF
									filter false
									colmake key:"test"
								EOF
							}
						}

						resource "observe_correlation_tag" "ctag_1" {
							column = "key"
							dataset = observe_dataset.b.oid
							name = "first_correlation_tag"
						}

						data "observe_dataset" "lookup_by_id" {
							id = observe_dataset.b.id
							depends_on = [observe_correlation_tag.ctag_1]
						}
					`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_dataset.lookup_by_id", "workspace"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "name", randomPrefix+"-b"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "stage.0.pipeline", "filter false\ncolmake key:\"test\"\n"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "correlation_tag.0.name", "first_correlation_tag"),
					resource.TestCheckResourceAttr("data.observe_dataset.lookup_by_id", "correlation_tag.0.column", "key"),
				),
			},
		},
	})
}
