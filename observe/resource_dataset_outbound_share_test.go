package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatasetOutboundShare(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
					resource "observe_snowflake_share_outbound" "test" {
						workspace   = data.observe_workspace.default.oid
						name        = "%[1]s"
						description = "test description"

						account {
							account = "io79077"
							organization = "HC83707"
						}
					}

					resource "observe_dataset" "test" {
						workspace = data.observe_workspace.default.oid
						name      = "%[1]s-ds"
	
						inputs = {
							"test" = observe_datastream.test.dataset
						}
	
						stage {}
					}

					resource "observe_dataset_outbound_share" "test" {
						workspace      = data.observe_workspace.default.oid
						description    = "test description"
						name 				   = "%[1]s"
						dataset        = observe_dataset.test.oid
						outbound_share = observe_snowflake_share_outbound.test.oid
						schema_name    = "%[1]s"
						view_name			 = "%[1]s"
						freshness_goal = "15m"
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_dataset_outbound_share.test", "workspace"),
					resource.TestCheckResourceAttrSet("observe_dataset_outbound_share.test", "oid"),
					resource.TestCheckResourceAttr("observe_dataset_outbound_share.test", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset_outbound_share.test", "description", "test description"),
					// TODO: implement custom TestCheckFunc that can compare OID without version
					// This OID has no version, while observe_dataset does, preventing direct comparison with TestCheckResourceAttrPair
					resource.TestCheckResourceAttrSet("observe_dataset_outbound_share.test", "dataset"),
					resource.TestCheckResourceAttrPair("observe_dataset_outbound_share.test", "outbound_share", "observe_snowflake_share_outbound.test", "oid"),
					resource.TestCheckResourceAttr("observe_dataset_outbound_share.test", "schema_name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset_outbound_share.test", "view_name", randomPrefix),
					resource.TestCheckResourceAttr("observe_dataset_outbound_share.test", "freshness_goal", "15m0s"),
				),
			},
		},
	})
}
