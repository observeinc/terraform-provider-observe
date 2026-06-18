package observe

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDatastreamNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_datastream" "no_ws" {
			name = "%s"
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: append(testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_datastream.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttrSet("observe_datastream.no_ws", "dataset"),
		), resource.TestStep{
			ResourceName:            "observe_datastream.no_ws",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"workspace"},
		}),
	})
}

func TestAccObserveIngestTokenNoWorkspace(t *testing.T) {
	config := `
		resource "observe_ingest_token" "no_ws" {}
	`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttrSet("observe_ingest_token.no_ws", "oid"),
			resource.TestCheckResourceAttrSet("observe_ingest_token.no_ws", "name"),
			resource.TestCheckResourceAttrSet("observe_ingest_token.no_ws", "secret"),
		),
	})
}

func TestAccObserveBookmarkGroupNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_bookmark_group" "no_ws" {
			name        = "%s"
			description = "no workspace test"
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_bookmark_group.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttrSet("observe_bookmark_group.no_ws", "oid"),
		),
	})
}

func TestAccObserveWorksheetNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_worksheet" "no_ws" {
			name     = "%s"
			icon_url = "test"
			queries = <<-EOF
			[{
				"id": "stage",
				"pipeline": "filter field = \"cpu_usage_core_seconds\"",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_worksheet.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveDropFilterNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_drop_filter" "no_ws" {
			name           = "%[1]s-filter"
			pipeline       = "filter FIELDS.x ~ y"
			source_dataset = observe_datastream.test_no_ws.dataset
			drop_rate      = 0.99
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_drop_filter.no_ws", "name", randomPrefix+"-filter"),
			resource.TestCheckResourceAttr("observe_drop_filter.no_ws", "drop_rate", "0.99"),
		),
	})
}

func TestAccObserveLinkNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(linkConfigNoWorkspacePreamble(randomPrefix)+`
		resource "observe_link" "no_ws" {
			source = observe_dataset.a.oid
			target = observe_dataset.b.oid
			fields = ["key:key"]
			label  = "%[1]s"
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_link.no_ws", "label", randomPrefix),
			resource.TestCheckResourceAttr("observe_link.no_ws", "fields.0", "key"),
		),
	})
}

func TestAccObserveMonitorNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_monitor" "no_ws" {
			name = "%[1]s"

			inputs = {
				"test" = observe_datastream.test_no_ws.dataset
			}

			stage {}

			rule {
				count {
					compare_function = "less_or_equal"
					compare_values   = [1]
					lookback_time    = "1m"
				}
			}
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_monitor.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorV2NoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_monitor_v2" "no_ws" {
			rule_kind     = "count"
			name          = "%[1]s"
			lookback_time = "30m"
			inputs = {
				"test" = observe_datastream.test_no_ws.dataset
			}
			stage {
				pipeline     = "colmake kind:\"test\", description:\"test\""
				output_stage = true
			}
			stage {
				pipeline = "filter kind ~ \"test\""
			}
			rules {
				level = "informational"
				count {
					compare_values {
						compare_fn  = "greater"
						value_int64 = [0]
					}
				}
			}
			scheduling {
				transform {
					freshness_goal = "15m"
				}
			}
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_monitor_v2.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttr("observe_monitor_v2.no_ws", "rule_kind", "count"),
		),
	})
}

func TestAccObserveMonitorActionNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_monitor_action" "no_ws" {
			name     = "%s"
			icon_url = "test"

			webhook {
				url_template  = "https://example.com"
				body_template = "{}"
			}
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_monitor_action.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveMonitorV2ActionNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_monitor_v2_action" "no_ws" {
			type = "webhook"
			name = "%s"
			webhook {
				url     = "https://example.com/"
				method  = "post"
				body    = "test body"
				fragments = jsonencode({ foo = "bar" })
				headers {
					header = "test"
					value  = "value"
				}
			}
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_monitor_v2_action.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttr("observe_monitor_v2_action.no_ws", "type", "webhook"),
		),
	})
}

func TestAccObserveMonitorActionAttachmentNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_monitor" "mon" {
			name = "%[1]s-mon"
			inputs = { "test" = observe_datastream.test_no_ws.dataset }
			stage {}
			rule {
				count {
					compare_function = "less_or_equal"
					compare_values   = [1]
					lookback_time    = "1m"
				}
			}
		}

		resource "observe_monitor_action" "act" {
			name = "%[1]s-act"
			webhook {
				url_template  = "https://example.com"
				body_template = "{}"
			}
		}

		resource "observe_monitor_action_attachment" "no_ws" {
			name    = "%[1]s-attach"
			monitor = observe_monitor.mon.oid
			action  = observe_monitor_action.act.oid
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_monitor_action_attachment.no_ws", "name", randomPrefix+"-attach"),
		),
	})
}

func TestAccObserveChannelNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_channel" "no_ws" {
			name     = "%s"
			icon_url = "test"
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_channel.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveChannelActionNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_channel_action" "no_ws" {
			name     = "%s"
			icon_url = "test"

			webhook {
				url     = "https://example.com"
				body    = "{}"
				headers = { "test" = "hello" }
			}
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_channel_action.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveLayeredSettingRecordNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_layered_setting_record" "no_ws" {
			name        = "%[1]s"
			setting     = "Scanner.powerLevel"
			value_int64 = 9009
			target      = observe_datastream.test_no_ws.dataset
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_layered_setting_record.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttr("observe_layered_setting_record.no_ws", "value_int64", "9009"),
		),
	})
}

func TestAccObservePollerNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_poller" "no_ws" {
			name       = "%[1]s-http"
			interval   = "1m"
			retries    = 5
			datastream = observe_datastream.test_no_ws.oid
			skip_external_validation = true

			http {
				method       = "POST"
				body         = jsonencode({ "hello" = "world" })
				endpoint     = "https://test.com"
				content_type = "application/json"
			}
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_poller.no_ws", "name", randomPrefix+"-http"),
			resource.TestCheckResourceAttr("observe_poller.no_ws", "kind", "HTTP"),
		),
	})
}

func TestAccObserveDashboardNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_dashboard" "no_ws" {
			name     = "%[1]s"
			icon_url = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter field = \"cpu_usage_core_seconds\"",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_dashboard.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveDashboardLinkNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_dashboard" "from" {
			name     = "%[1]s-from"
			icon_url = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}

		resource "observe_dashboard" "to" {
			name     = "%[1]s-to"
			icon_url = "test"
			stages = <<-EOF
			[{
				"pipeline": "filter true",
				"input": [{
					"inputName": "kubernetes/metrics/Container Metrics",
					"inputRole": "Data",
					"datasetId": "41042989"
				}]
			}]
			EOF
		}

		resource "observe_dashboard_link" "no_ws" {
			name           = "%[1]s-link"
			description    = "no workspace link"
			from_dashboard = observe_dashboard.from.oid
			to_dashboard   = observe_dashboard.to.oid
			link_label     = "link"
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_dashboard_link.no_ws", "name", randomPrefix+"-link"),
			resource.TestCheckResourceAttr("observe_dashboard_link.no_ws", "link_label", "link"),
		),
	})
}

func TestAccObservePreferredPathNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(linkConfigNoWorkspacePreamble(randomPrefix)+`
		resource "observe_link" "a_to_b" {
			source = observe_dataset.a.oid
			target = observe_dataset.b.oid
			fields = ["key:key"]
			label  = "to_b"
		}

		resource "observe_link" "b_to_a" {
			source = observe_dataset.b.oid
			target = observe_dataset.a.oid
			fields = ["key:key"]
			label  = "to_a"
		}

		resource "observe_folder" "folder" {
			name = "%[1]s-folder"
		}

		resource "observe_preferred_path" "no_ws" {
			folder      = observe_folder.folder.oid
			name        = "%[1]s-path"
			description = "no workspace path"
			source      = observe_dataset.a.oid

			step { link = observe_link.a_to_b.oid }
			step { link = observe_link.b_to_a.oid }
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_preferred_path.no_ws", "name", randomPrefix+"-path"),
		),
	})
}

func TestAccObserveSnowflakeOutboundShareNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_snowflake_outbound_share" "no_ws" {
			name        = "%[1]s"
			description = "no workspace test"

			account {
				account      = "io79077"
				organization = "HC83707"
			}
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_snowflake_outbound_share.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttrSet("observe_snowflake_outbound_share.no_ws", "oid"),
		),
	})
}

func TestAccObserveDatasetOutboundShareNoWorkspace(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_snowflake_outbound_share" "share" {
			name        = "%[1]s-share"
			description = "no workspace test"

			account {
				account      = "io79077"
				organization = "HC83707"
			}
		}

		resource "observe_dataset" "ds" {
			name = "%[1]s-ds"
			inputs = { "test" = observe_datastream.test_no_ws.dataset }
			stage {}
		}

		resource "observe_dataset_outbound_share" "no_ws" {
			name           = "%[1]s"
			dataset        = observe_dataset.ds.oid
			outbound_share = observe_snowflake_outbound_share.share.oid
			schema_name    = "%[1]s"
			view_name      = "%[1]s"
			freshness_goal = "15m"
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_dataset_outbound_share.no_ws", "name", randomPrefix),
			resource.TestCheckResourceAttrSet("observe_dataset_outbound_share.no_ws", "oid"),
		),
	})
}

func TestAccObserveSourceDatasetNoWorkspace(t *testing.T) {
	if os.Getenv("CI") != "true" {
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's Snowflake database.")
	}

	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_source_dataset" "no_ws" {
			name                     = "%[1]s"
			schema                   = "EXTERNAL"
			table_name               = "%[1]s_TABLE_NAME"
			source_update_table_name = "%[1]s_SOURCE_UPDATE_TABLE_NAME"
			valid_from_field         = "TIMESTAMP"

			field {
				name     = "TIMESTAMP"
				type     = "timestamp"
				sql_type = "NUMBER(38,0)"
			}
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_source_dataset.no_ws", "name", randomPrefix),
		),
	})
}

func TestAccObserveFiledropNoWorkspace(t *testing.T) {
	filedropRoleArn := os.Getenv("OBSERVE_FILEDROP_ROLE_ARN")
	if os.Getenv("CI") != "true" {
		t.Skip("CI != true. This test requires manual setup that has only been performed on the CI account's AWS account.")
	}

	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_filedrop" "no_ws" {
			datastream = observe_datastream.test_no_ws.oid
			config {
				provider {
					aws {
						region   = "us-west-2"
						role_arn = "%[2]s"
					}
				}
			}
		}`, randomPrefix, filedropRoleArn)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttrSet("observe_filedrop.no_ws", "name"),
			resource.TestCheckResourceAttrSet("observe_filedrop.no_ws", "status"),
		),
	})
}

func TestAccObserveDatasetNoWorkspaceImport(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(datastreamNoWorkspacePreamble+`
		resource "observe_dataset" "no_ws" {
			name = "%[1]s-no-ws"

			inputs = {
				"test" = observe_datastream.test_no_ws.dataset
			}

			stage {}
		}`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: append(testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_dataset.no_ws", "name", randomPrefix+"-no-ws"),
		), resource.TestStep{
			ResourceName:            "observe_dataset.no_ws",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"workspace"},
		}),
	})
}

func TestAccObserveFolderNoWorkspaceImport(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")
	config := fmt.Sprintf(`
		resource "observe_folder" "no_ws" {
			name     = "%s"
			icon_url = "test"
		}
	`, randomPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: append(testAccNoWorkspaceSteps(config,
			resource.TestCheckResourceAttr("observe_folder.no_ws", "name", randomPrefix),
		), resource.TestStep{
			ResourceName:            "observe_folder.no_ws",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"workspace"},
		}),
	})
}
