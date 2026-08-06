package observe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/observeinc/terraform-provider-observe/client/binding"
)

func TestAccObserveMonitorV2ActionEmailDatasource(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					data "observe_user" "system" {
						email = "%[2]s"
					}

					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "somebody once told me"
							body = "the world is gonna roll me"
							fragments = jsonencode({
								foo = "bar"
							})
							addresses = ["test@observeinc.com"]
							users = [data.observe_user.system.oid]
						}
						name = "%[1]s"
						description = "an interesting description"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "type", "email"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.subject", "somebody once told me"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.body", "the world is gonna roll me"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.addresses.0", "test@observeinc.com"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "email.0.users.#", "1"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2ActionWebhookDatasource(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(monitorV2ConfigPreamble+`
					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "webhook"
						webhook {
							headers {
								header = "never gonna give you up"
								value = "never gonna let you down"
							}
							body = "never gonna run around and desert you"
							fragments = jsonencode({
								foo = "bar"
							})
							url = "https://example.com/"
							method = "post"
						}
						name = "%[1]s"
						description = "an interesting description"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_monitor_v2_action.act", "workspace"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "name", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "type", "webhook"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "description", "an interesting description"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.fragments", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.headers.0.header", "never gonna give you up"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.headers.0.value", "never gonna let you down"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.body", "never gonna run around and desert you"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.url", "https://example.com/"),
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "webhook.0.method", "post"),
				),
			},
		},
	})
}

func TestAccObserveMonitorV2ActionExportWithBindings(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	providerPreamble := `
		terraform {} # trick the testing framework into not mangling our config
		provider "observe" {
			export_object_bindings = true
		}
	`

	workspaceTfName := fmt.Sprintf("workspace_%s", strings.ToLower(defaultWorkspaceName))
	workspaceTfLocalBindingVar := fmt.Sprintf("binding__monitor_v2_action_%s__%s", randomPrefix, workspaceTfName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// email action with a user OID — expect workspace + user bindings
				Config: fmt.Sprintf(providerPreamble+monitorV2ConfigPreamble+`
					data "observe_user" "system" {
						email = "%[2]s"
					}

					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "email"
						email {
							subject = "export bindings test"
							body    = "body"
							users   = [data.observe_user.system.oid]
						}
						name = "%[1]s"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix, systemUser()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "workspace", fmt.Sprintf("${local.%s}", workspaceTfLocalBindingVar)),
					resource.TestCheckResourceAttrWith("data.observe_monitor_v2_action.act", "_bindings", func(value string) error {
						var bindings binding.BindingsObject
						if err := json.Unmarshal([]byte(value), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindUser, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kinds does not match: expected %#v, got %#v", expectedKinds, bindings.Kinds)
						}
						expectedWorkspaceBinding := binding.Target{TfLocalBindingVar: workspaceTfLocalBindingVar, TfName: workspaceTfName, IsOid: true}
						if bindings.Workspace != expectedWorkspaceBinding {
							return fmt.Errorf("bindings.Workspace does not match: expected %#v, got %#v", expectedWorkspaceBinding, bindings.Workspace)
						}
						// user binding keyed by email
						userRef := binding.Ref{Kind: binding.KindUser, Key: systemUser()}
						if _, ok := bindings.Mappings[userRef]; !ok {
							return fmt.Errorf("expected user binding with key %q, got mappings: %#v", systemUser(), bindings.Mappings)
						}
						return nil
					}),
				),
			},
			{
				// webhook action — only workspace binding, no user OIDs
				Config: fmt.Sprintf(providerPreamble+monitorV2ConfigPreamble+`
					resource "observe_monitor_v2_action" "act" {
						workspace = data.observe_workspace.default.oid
						type = "webhook"
						webhook {
							url    = "https://example.com/"
							method = "post"
							body   = "test"
						}
						name = "%[1]s_wh"
					}

					data "observe_monitor_v2_action" "act" {
						id = observe_monitor_v2_action.act.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_monitor_v2_action.act", "workspace",
						fmt.Sprintf("${local.binding__monitor_v2_action_%s_wh__%s}", randomPrefix, workspaceTfName)),
					resource.TestCheckResourceAttrWith("data.observe_monitor_v2_action.act", "_bindings", func(value string) error {
						var bindings binding.BindingsObject
						if err := json.Unmarshal([]byte(value), &bindings); err != nil {
							return err
						}
						expectedKinds := []binding.Kind{binding.KindUser, binding.KindWorkspace}
						if !reflect.DeepEqual(bindings.Kinds, expectedKinds) {
							return fmt.Errorf("bindings.Kinds does not match: expected %#v, got %#v", expectedKinds, bindings.Kinds)
						}
						// no user OIDs in webhook action, so Mappings should only have workspace
						for ref := range bindings.Mappings {
							if ref.Kind == binding.KindUser {
								return fmt.Errorf("unexpected user binding in webhook action: %#v", ref)
							}
						}
						return nil
					}),
				),
			},
		},
	})
}
