package observe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func TestCompositeMonitorCreateRequest(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceCompositeMonitor().Schema, map[string]interface{}{
		"name":      "both down",
		"condition": "A && !B",
		"constituents": []interface{}{
			map[string]interface{}{"alias": "A", "monitor": "o:::monitorv2:41000001"},
			map[string]interface{}{"alias": "B", "monitor": "o:::monitorv2:41000002", "notify_independently": false},
		},
		"actions": []interface{}{
			map[string]interface{}{
				"oid":    "o:::monitorv2action:41000300",
				"levels": []interface{}{"critical", "error"},
			},
			map[string]interface{}{
				"action": []interface{}{map[string]interface{}{
					"type": "webhook",
					"webhook": []interface{}{map[string]interface{}{
						"url":    "https://example.com/hook",
						"method": "post",
						"body":   "{}",
					}},
				}},
				"send_end_notifications":  true,
				"send_reminders_interval": "10m",
				"conditions": []interface{}{map[string]interface{}{
					"compare_terms": []interface{}{map[string]interface{}{
						"comparison": []interface{}{map[string]interface{}{"compare_fn": "equal", "value_string": []interface{}{"web-1"}}},
						"column":     []interface{}{map[string]interface{}{"correlation_tag": []interface{}{map[string]interface{}{"tag": "host"}}}},
					}},
				}},
			},
		},
		"custom_variables": map[string]interface{}{"title": "down"},
	})

	req, diags := newCompositeMonitorCreateRequest(data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	got, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"label":"both down","disabled":false,` +
		`"constituents":[{"alias":"A","monitor":{"id":"41000001"},"notifyIndependently":true},{"alias":"B","monitor":{"id":"41000002"},"notifyIndependently":false}],` +
		`"condition":"A \u0026\u0026 !B",` +
		`"actions":[{"action":{"id":"41000300"},"levels":["Critical","Error"]},` +
		`{"inlineAction":{"type":"Webhook","webhook":{"url":"https://example.com/hook","method":"Post","body":"{}"}},` +
		`"conditions":{"operator":"And","compareTerms":[{"tag":"host","comparison":{"compareFn":"Equal","compareValue":"web-1","valueType":"String"}}]},` +
		`"sendEndNotifications":true,"sendRemindersIntervalSeconds":600}],` +
		`"customVariables":{"title":"down"}}`
	if string(got) != want {
		t.Errorf("create request:\n got %s\nwant %s", got, want)
	}
}

func TestCompositeMonitorInlineActionDescriptionRejected(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceCompositeMonitor().Schema, map[string]interface{}{
		"name":      "x",
		"condition": "A && B",
		"constituents": []interface{}{
			map[string]interface{}{"alias": "A", "monitor": "o:::monitorv2:1"},
			map[string]interface{}{"alias": "B", "monitor": "o:::monitorv2:2"},
		},
		"actions": []interface{}{map[string]interface{}{
			"action": []interface{}{map[string]interface{}{
				"type":        "email",
				"description": "not supported",
				"email":       []interface{}{map[string]interface{}{"subject": "s", "addresses": []interface{}{"a@example.com"}}},
			}},
		}},
	})
	if _, diags := newCompositeMonitorCreateRequest(data); !diags.HasError() {
		t.Fatal("expected an error for an inline action description")
	}
}

func TestCompositeMonitorComparisonRoundTrip(t *testing.T) {
	b := true
	i := types.Int64Scalar(42)
	f := 1.5
	s := "web-1"
	d := types.Int64Scalar(90_000_000_000)
	for name, tc := range map[string]struct {
		value gql.PrimitiveValueInput
		key   string
		want  interface{}
	}{
		"bool":     {gql.PrimitiveValueInput{Bool: &b}, "value_bool", true},
		"int64":    {gql.PrimitiveValueInput{Int64: &i}, "value_int64", 42},
		"float64":  {gql.PrimitiveValueInput{Float64: &f}, "value_float64", 1.5},
		"string":   {gql.PrimitiveValueInput{String: &s}, "value_string", "web-1"},
		"duration": {gql.PrimitiveValueInput{Duration: &d}, "value_duration", "1m30s"},
	} {
		t.Run(name, func(t *testing.T) {
			in := compositeMonitorComparisonFromGQL(gql.MonitorV2ComparisonInput{CompareFn: "Greater", CompareValue: tc.value})
			out, err := compositeMonitorFlattenComparison(rest.MonitorComparison{CompareFn: in.CompareFn, CompareValue: in.CompareValue, ValueType: in.ValueType})
			if err != nil {
				t.Fatal(err)
			}
			if out["compare_fn"] != "greater" {
				t.Errorf("compare_fn = %v", out["compare_fn"])
			}
			if got := out[tc.key].([]interface{}); len(got) != 1 || !reflect.DeepEqual(got[0], tc.want) {
				t.Errorf("%s = %#v, want %#v", tc.key, got, tc.want)
			}
		})
	}
}

func TestCompositeMonitorToResourceData(t *testing.T) {
	var m rest.CompositeMonitorResource
	if err := json.Unmarshal([]byte(`{
		"id": "41000100", "label": "both down", "description": "desc", "disabled": true,
		"constituents": [
			{"alias": "A", "monitor": {"id": "41000001"}, "notifyIndependently": true},
			{"alias": "B", "monitor": {"id": "41000002"}, "notifyIndependently": false}
		],
		"condition": "A && B",
		"groupings": [{"linkColumn": null, "columnPath": null, "correlationTag": {"tag": "host"}}],
		"evaluationIntervalSeconds": 60,
		"actions": [
			{"action": {"id": "41000300", "record": {"label": "shared", "type": "Email", "inline": false}},
			 "levels": [], "conditions": null, "sendEndNotifications": false, "sendRemindersIntervalSeconds": null},
			{"action": {"id": "41000200", "record": {"label": null, "type": "Email", "inline": true, "webhook": null,
				"email": {"users": [{"id": "7"}], "addresses": ["a@example.com"], "subject": "s", "body": null, "fragments": {"k": "v"}}}},
			 "levels": ["NoData"], "conditions": {"operator": "Or", "subExpressions": [], "compareTerms": [
				{"column": "svc", "path": "a.b", "link": null, "tag": null, "comparison": {"compareFn": "IsNull", "compareValue": null, "valueType": null}}]},
			 "sendEndNotifications": true, "sendRemindersIntervalSeconds": 600}
		],
		"customVariables": {"title": "down"}
	}`), &m); err != nil {
		t.Fatal(err)
	}

	data := schema.TestResourceDataRaw(t, resourceCompositeMonitor().Schema, map[string]interface{}{})
	if diags := compositeMonitorToResourceData(&m, data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	for key, want := range map[string]interface{}{
		"oid":                                  "o:::monitorv2:41000100",
		"name":                                 "both down",
		"description":                          "desc",
		"disabled":                             true,
		"constituents.1.monitor":               "o:::monitorv2:41000002",
		"constituents.1.notify_independently":  false,
		"groupings.0.correlation_tag.0.tag":    "host",
		"evaluation_interval":                  "1m0s",
		"actions.0.oid":                        "o:::monitorv2action:41000300",
		"actions.1.inline_action_id":           "41000200",
		"actions.1.action.0.type":              "email",
		"actions.1.action.0.email.0.users.0":   "o:::user:7",
		"actions.1.action.0.email.0.fragments": `{"k": "v"}`,
		"actions.1.levels.0":                   "no_data",
		"actions.1.send_reminders_interval":    "10m0s",
		"actions.1.conditions.0.operator":      "or",
		"actions.1.conditions.0.compare_terms.0.column.0.column_path.0.path": "a.b",
		"actions.1.conditions.0.compare_terms.0.comparison.0.compare_fn":     "is_null",
		"custom_variables.title": "down",
	} {
		if got := data.Get(key); !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

// The composite monitors API must be enabled (MonitorV2.compositeRestApi) for the
// customer running acceptance tests.
func compositeMonitorConstituentsConfig(prefix string) string {
	constituent := func(name string) string {
		return fmt.Sprintf(`
		resource "observe_monitor_v2" "%[2]s" {
			workspace     = data.observe_workspace.default.oid
			rule_kind     = "count"
			name          = "%[1]s-%[2]s"
			lookback_time = "30m"
			inputs = {
				"test" = observe_datastream.test.dataset
			}
			stage {
				pipeline = <<-EOF
					colmake kind:"test", description:"test"
				EOF
				output_stage = true
			}
			rules {
				level = "warning"
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
		}
		`, prefix, name)
	}
	return fmt.Sprintf(monitorV2ConfigPreamble, prefix) + constituent("a") + constituent("b") + fmt.Sprintf(`
		resource "observe_monitor_v2_action" "shared" {
			workspace = data.observe_workspace.default.oid
			type      = "email"
			name      = "%[1]s-shared"
			email {
				subject   = "composite"
				addresses = ["test@observeinc.com"]
			}
		}
	`, prefix)
}

func TestAccObserveCompositeMonitor(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: compositeMonitorConstituentsConfig(randomPrefix) + fmt.Sprintf(`
					resource "observe_composite_monitor" "both" {
						name        = "%[1]s-both"
						description = "both constituents alerting"
						condition   = "A && B"
						constituents {
							alias   = "A"
							monitor = observe_monitor_v2.a.oid
						}
						constituents {
							alias                = "B"
							monitor              = observe_monitor_v2.b.oid
							notify_independently = false
						}
						actions {
							oid    = observe_monitor_v2_action.shared.oid
							levels = ["warning"]
						}
						actions {
							action {
								type = "webhook"
								webhook {
									url    = "https://example.com/hook"
									method = "post"
									body   = "{}"
								}
							}
							send_end_notifications  = true
							send_reminders_interval = "10m"
						}
						custom_variables = {
							title = "both down"
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_composite_monitor.both", "oid"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "name", randomPrefix+"-both"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "condition", "A && B"),
					resource.TestCheckResourceAttrPair("observe_composite_monitor.both", "constituents.0.monitor", "observe_monitor_v2.a", "oid"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "constituents.0.notify_independently", "true"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "constituents.1.notify_independently", "false"),
					resource.TestCheckResourceAttrPair("observe_composite_monitor.both", "actions.0.oid", "observe_monitor_v2_action.shared", "oid"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "actions.0.levels.0", "warning"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "actions.1.action.0.webhook.0.url", "https://example.com/hook"),
					resource.TestCheckResourceAttrSet("observe_composite_monitor.both", "actions.1.inline_action_id"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "actions.1.send_reminders_interval", "10m0s"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "custom_variables.title", "both down"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "groupings.#", "0"),
				),
			},
			{
				Config: compositeMonitorConstituentsConfig(randomPrefix) + fmt.Sprintf(`
					resource "observe_composite_monitor" "both" {
						name      = "%[1]s-either"
						condition = "A || !B"
						disabled  = true
						constituents {
							alias   = "A"
							monitor = observe_monitor_v2.a.oid
						}
						constituents {
							alias   = "B"
							monitor = observe_monitor_v2.b.oid
						}
						actions {
							action {
								type = "webhook"
								webhook {
									url    = "https://example.com/other"
									method = "put"
									body   = "{}"
								}
							}
						}
						custom_variables = {
							message = "one is down"
						}
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "name", randomPrefix+"-either"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "description", ""),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "condition", "A || !B"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "constituents.1.notify_independently", "true"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "actions.#", "1"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "actions.0.action.0.webhook.0.method", "put"),
					resource.TestCheckNoResourceAttr("observe_composite_monitor.both", "custom_variables.title"),
					resource.TestCheckResourceAttr("observe_composite_monitor.both", "custom_variables.message", "one is down"),
				),
			},
			{
				ResourceName:      "observe_composite_monitor.both",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
