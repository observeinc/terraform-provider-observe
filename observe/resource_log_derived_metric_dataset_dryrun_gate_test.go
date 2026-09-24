package observe

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// baseLogDerivedMetricState is a populated state for one log-derived metric dataset, so a
// matching config leaves the validated attributes alone and the tests measure only what they
// intend to.
func baseLogDerivedMetricState(shapingQuery string) map[string]string {
	return map[string]string{
		"id":                         "41000005",
		"oid":                        "o:::dataset:41000005",
		"workspace":                  "o:::workspace:41000215",
		"metric_name":                "my_metric",
		"metric_type":                "",
		"unit":                       "",
		"interval":                   "",
		"description":                "",
		"icon_url":                   "",
		"input":                      "o:::dataset:41000002",
		"shaping_query":              shapingQuery,
		"aggregation.#":              "1",
		"aggregation.0.function":     "SUM",
		"aggregation.0.field_path.#": "0",
		"metric_tag.#":               "0",
	}
}

func baseLogDerivedMetricConfig(shapingQuery string) map[string]interface{} {
	return map[string]interface{}{
		"workspace":     "o:::workspace:41000215",
		"metric_name":   "my_metric",
		"input":         "o:::dataset:41000002",
		"shaping_query": shapingQuery,
		"aggregation": []interface{}{
			map[string]interface{}{"function": "SUM"},
		},
	}
}

// TestLogDerivedMetricDryRunGate is the log-derived-metric twin of
// TestDatasetDryRunGateIgnoresSuppressedDiffs / TestGateNeverSkipsAValidationRelevantUpdate.
//
// It is not a mechanical duplicate. This resource gates on different attributes (input,
// shaping_query, metric_name), and shaping_query carries diffSuppressPipeline -- a TypeString
// with the same trailing-whitespace suppressor the dataset's stage pipelines use. So the exact
// bug being fixed is reachable here: a shaping_query differing only in trailing whitespace
// renders no plan, yet d.HasChange("shaping_query") reports a change, and the old gate paid a
// dry-run save per resource for it.
//
// Both directions of the invariant are asserted, as for datasets:
//
//	gate skips validation  =>  the plan updates nothing under any validated key
//	gate validates         =>  the plan updates something under a validated key (except creates)
func TestLogDerivedMetricDryRunGate(t *testing.T) {
	cases := []struct {
		name  string
		state map[string]string
		cfg   map[string]interface{}

		// what the old, suppression-blind gate decided
		wantHasChange bool
		// what the shared production gate decides
		wantValidates bool
	}{
		{
			// THE BUG, on this resource: trailing whitespace only, forgiven by
			// diffSuppressPipeline, so nothing is planned -- but HasChange still fires.
			name:          "suppressed_only_trailing_whitespace",
			state:         baseLogDerivedMetricState("filter true"),
			cfg:           baseLogDerivedMetricConfig("filter true\n    "),
			wantHasChange: true,
			wantValidates: false,
		},
		{
			name:          "identical",
			state:         baseLogDerivedMetricState("filter true"),
			cfg:           baseLogDerivedMetricConfig("filter true"),
			wantHasChange: false,
			wantValidates: false,
		},
		{
			name:          "real_shaping_query_change",
			state:         baseLogDerivedMetricState("filter true"),
			cfg:           baseLogDerivedMetricConfig("filter false"),
			wantHasChange: true,
			wantValidates: true,
		},
		{
			name:  "real_input_change",
			state: baseLogDerivedMetricState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseLogDerivedMetricConfig("filter true")
				c["input"] = "o:::dataset:41000099"
				return c
			}(),
			wantHasChange: true,
			wantValidates: true,
		},
		{
			name:  "real_metric_name_change",
			state: baseLogDerivedMetricState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseLogDerivedMetricConfig("filter true")
				c["metric_name"] = "renamed_metric"
				return c
			}(),
			wantHasChange: true,
			wantValidates: true,
		},
		{
			// An update confined to an attribute the backend need not re-validate.
			name:  "unvalidated_attribute_only",
			state: baseLogDerivedMetricState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseLogDerivedMetricConfig("filter true")
				c["description"] = "changed"
				return c
			}(),
			wantHasChange: false,
			wantValidates: false,
		},
		{
			name:          "create",
			state:         map[string]string{},
			cfg:           baseLogDerivedMetricConfig("filter true"),
			wantHasChange: true,
			wantValidates: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := resourceLogDerivedMetricDataset()

			var hasChange, validates bool
			r.CustomizeDiff = func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
				hasChange = d.HasChange("input") || d.HasChange("shaping_query") || d.HasChange("metric_name")
				// The production gate itself, shared with validateLogDerivedMetricDatasetChanges.
				validates = needsDryRunValidation(d, logDerivedMetricValidationKeys)
				return nil
			}

			state := &terraform.InstanceState{ID: tc.state["id"], Attributes: tc.state}
			diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(tc.cfg), nil)
			if err != nil {
				t.Fatalf("Diff error: %v", err)
			}

			updatedValidated := []string{}
			if diff != nil {
				for k := range diff.Attributes {
					for _, key := range logDerivedMetricValidationKeys {
						if k == key || strings.HasPrefix(k, key+".") {
							updatedValidated = append(updatedValidated, k)
							break
						}
					}
				}
			}
			sort.Strings(updatedValidated)

			if hasChange != tc.wantHasChange {
				t.Errorf("d.HasChange gate = %v, want %v", hasChange, tc.wantHasChange)
			}
			if validates != tc.wantValidates {
				t.Errorf("needsDryRunValidation = %v, want %v", validates, tc.wantValidates)
			}

			// The invariant, both directions.
			if !validates && len(updatedValidated) > 0 {
				t.Errorf("gate skipped validation, but terraform will update validated attributes %v -- "+
					"an unvalidated change would reach the backend", updatedValidated)
			}
			if validates && len(updatedValidated) == 0 && state.ID != "" {
				t.Errorf("gate would validate, but terraform updates nothing under %v -- "+
					"this is the wasted-preflight bug returning", logDerivedMetricValidationKeys)
			}
			t.Logf("validates=%-5v updatesValidatedAttrs=%v", validates, updatedValidated)
		})
	}
}
