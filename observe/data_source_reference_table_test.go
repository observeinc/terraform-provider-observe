package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDataSourceReferenceTable(t *testing.T) {
	t.Skip()
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_reference_table" "test" {
					label      = "%[1]s"
					source_file = "testdata/reference_table.csv"
					checksum = filemd5("testdata/reference_table.csv")
					description = "test"
					primary_key = ["state_code"]
					label_field = "state"
				}

				data "observe_reference_table" "by_id" {
					id = observe_reference_table.test.id
				}

				data "observe_reference_table" "by_label" {
					label = "%[1]s"
					
					// need explicit dependency since the reference table needs to be created before
					// we can lookup, and we're (deliberately) not doing label = observe_reference_table.test.label
					depends_on = [observe_reference_table.test]
				}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_reference_table.by_id", "label", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_reference_table.by_label", "label", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_reference_table.by_id", "description", "test"),
					resource.TestCheckResourceAttr("data.observe_reference_table.by_label", "description", "test"),
					resource.TestCheckResourceAttrSet("data.observe_reference_table.by_id", "dataset"),
					resource.TestCheckResourceAttrSet("data.observe_reference_table.by_label", "dataset"),
					resource.TestCheckResourceAttr("data.observe_reference_table.by_id", "checksum", "93dc3e9f2c6e30cd956eb062c18112eb"),
					resource.TestCheckResourceAttr("data.observe_reference_table.by_label", "checksum", "93dc3e9f2c6e30cd956eb062c18112eb"),
					// TODO: add checks for primary_key and label_field once they're supported
				),
			},
		},
	})
}
