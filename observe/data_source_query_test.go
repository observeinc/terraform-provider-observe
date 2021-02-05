package observe

import (
	"fmt"
	"io/ioutil"
	"os"
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

func TestAccObserveSourceQueryPoll(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
					resource "observe_http_post" "test" {
					  data   = jsonencode({ hello = "world" })
					  tags = {
						"tf_test_id" = "%[1]s"
					  }
					}

					data "observe_query" "query" {
					  start = timeadd(timestamp(), "-10m")

					  inputs = { "observation" = data.observe_dataset.observation.oid }

					  stage {
						pipeline = <<-EOF
						  filter string(EXTRA.tf_test_id) = "%[1]s"
						EOF
					  }

					  poll {}
				  }
				`,
					randomPrefix,
				),
			},
		},
	})
}

func TestAccObserveSourceQueryAssert(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	golden_file, err := ioutil.TempFile("", "tf-assert")
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	defer os.Remove(golden_file.Name()) // clean up

	/* We will run a single plan which:
	- posts an observation
	- waits for the result
	- compares the result against a golden file

	We run this plan three times:
	- in the first run, the golden file is empty, so we expect it to fail
	- in a second run, we set the update flag to write to golden file
	- in a third run, the result should match the golden file
	*/
	tf_plan := fmt.Sprintf(configPreamble+`
		resource "observe_http_post" "test" {
		  data   = jsonencode({ hello = "world" })
		  tags = {
			"tf_test_id" = "%[1]s"
		  }
		}

		data "observe_query" "query" {
		  start = timeadd(timestamp(), "-10m")

		  inputs = { "observation" = data.observe_dataset.observation.oid }

		  stage {
			pipeline = <<-EOF
			  filter string(EXTRA.tf_test_id) = "%[1]s"
			EOF
		  }

		  poll {}

		  assert {
			update      = %%s		# Test will toggle this
			golden_file = "%[2]s"
		  }
	  }`, randomPrefix, golden_file.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(tf_plan, "false"),
				ExpectError: regexp.MustCompile("query result does not match golden file"),
			},
			{
				Config: fmt.Sprintf(tf_plan, "true"),
			},
			{
				Config: fmt.Sprintf(tf_plan, "false"),
			},
		},
	})
}