package observe

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// datasetGateVerdict runs the real dataset schema's diff machinery over a (state, config)
// pair and reports what the two candidate gates would decide, plus whether the plan renders
// any change at all.
//
// It swaps in its own CustomizeDiff so no backend client is needed: the point is which gate
// fires, not what the dry run would answer.
func datasetGateVerdict(t *testing.T, stateAttrs map[string]string, cfg map[string]interface{}) (planEmpty, hasChange, touches bool) {
	t.Helper()

	r := resourceDataset()
	r.CustomizeDiff = func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
		hasChange = d.HasChange("inputs") || d.HasChange("stage") || d.HasChange("name")
		// The production gate itself, not a copy of it.
		touches = needsDryRunValidation(d, datasetValidationKeys)
		return nil
	}

	state := &terraform.InstanceState{ID: stateAttrs["id"], Attributes: stateAttrs}
	diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(cfg), nil)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	planEmpty = diff == nil || diff.Empty()
	if !planEmpty {
		for k, v := range diff.Attributes {
			t.Logf("  rendered diff %q: %q -> %q", k, v.Old, v.New)
		}
	}
	return planEmpty, hasChange, touches
}

// baseDatasetState is a fully-populated state for one single-stage dataset. Every attribute
// the schema would otherwise default is set explicitly, so a matching config produces a
// genuinely empty plan and the tests below measure only what they intend to.
func baseDatasetState(pipeline string) map[string]string {
	return map[string]string{
		"id":                           "41000001",
		"oid":                          "o:::dataset:41000001",
		"workspace":                    "o:::workspace:41000215",
		"name":                         "ds",
		"acceleration_disabled":        "false",
		"acceleration_disabled_source": "",
		"inputs.%":                     "1",
		"inputs.test":                  "o:::dataset:41000002",
		"stage.#":                      "1",
		"stage.0.alias":                "",
		"stage.0.input":                "test",
		"stage.0.pipeline":             pipeline,
		"stage.0.output_stage":         "false",
	}
}

func baseDatasetConfig(pipeline string) map[string]interface{} {
	return map[string]interface{}{
		"workspace": "o:::workspace:41000215",
		"name":      "ds",
		"inputs":    map[string]interface{}{"test": "o:::dataset:41000002"},
		"stage": []interface{}{
			map[string]interface{}{"input": "test", "pipeline": pipeline},
		},
	}
}

// TestDatasetDryRunGateIgnoresSuppressedDiffs is the regression test for the no-op plan that
// still issues a dry-run save per dataset.
//
// validateDatasetChanges gates its dry run on whether inputs/stage/name changed. It used to
// ask d.HasChange, which does not consult the diff: it takes "old" from state and "new" from a
// merge that includes the config level, so it returns true for any difference a
// DiffSuppressFunc hides. stage carries two such suppressors today -- pipeline (trailing
// whitespace) and alias (last stage) -- and both are exercised below. The observable result on a
// large tenant was a terraform plan that reported "No changes. Your infrastructure matches the
// configuration." after 18.7 minutes, having issued ~782 dry runs -- one per managed dataset.
//
// The "suppressed_only" case below is that bug: the plan is empty, HasChange says true, and
// only the diff-based gate gets it right. The remaining cases pin that the gate still fires
// for changes that are real, so validation is not lost.
func TestDatasetDryRunGateIgnoresSuppressedDiffs(t *testing.T) {
	cases := []struct {
		name  string
		state map[string]string
		cfg   map[string]interface{}

		wantPlanEmpty bool
		// what the old HasChange-based gate decided
		wantHasChange bool
		// what the diff-based gate decides
		wantTouches bool
	}{
		{
			// THE BUG. Config carries the trailing "\n    " a <<EOT heredoc leaves behind;
			// diffSuppressPipeline forgives it, so the plan is empty -- but HasChange sees
			// the raw config value and reports a change, so a dry run fired anyway.
			name:          "suppressed_only_trailing_whitespace",
			state:         baseDatasetState("filter true"),
			cfg:           baseDatasetConfig("filter true\n    "),
			wantPlanEmpty: true,
			wantHasChange: true,
			wantTouches:   false,
		},
		{
			// The second suppressor on stage: alias on the LAST stage is suppressed
			// (flattenQuery cannot return it, so it would otherwise diff forever). With a
			// single stage, index 0 IS the last stage. Same shape as the whitespace case:
			// plan empty, HasChange true, diff-based gate false.
			name:  "suppressed_only_last_stage_alias",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["stage"] = []interface{}{
					map[string]interface{}{"input": "test", "pipeline": "filter true", "alias": "ignored"},
				}
				return c
			}(),
			wantPlanEmpty: true,
			wantHasChange: true,
			wantTouches:   false,
		},
		{
			// Control: byte-identical. Neither gate should fire.
			name:          "identical",
			state:         baseDatasetState("filter true"),
			cfg:           baseDatasetConfig("filter true"),
			wantPlanEmpty: true,
			wantHasChange: false,
			wantTouches:   false,
		},
		{
			// A real pipeline edit must still be validated.
			name:          "real_pipeline_change",
			state:         baseDatasetState("filter true"),
			cfg:           baseDatasetConfig("filter false"),
			wantPlanEmpty: false,
			wantHasChange: true,
			wantTouches:   true,
		},
		{
			// name is gated because we enforce uniqueness server-side.
			name:  "real_name_change",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["name"] = "renamed"
				return c
			}(),
			wantPlanEmpty: false,
			wantHasChange: true,
			wantTouches:   true,
		},
		{
			// Repointing an input changes what the pipeline resolves against.
			name:  "real_inputs_change",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["inputs"] = map[string]interface{}{"test": "o:::dataset:41000099"}
				return c
			}(),
			wantPlanEmpty: false,
			wantHasChange: true,
			wantTouches:   true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			planEmpty, hasChange, touches := datasetGateVerdict(t, tc.state, tc.cfg)

			if planEmpty != tc.wantPlanEmpty {
				t.Errorf("rendered plan empty = %v, want %v", planEmpty, tc.wantPlanEmpty)
			}
			// Pinned to document the SDK behaviour the fix works around. If a future SDK
			// makes HasChange suppression-aware this will fail, and diffTouchesAny can then
			// be reconsidered.
			if hasChange != tc.wantHasChange {
				t.Errorf("d.HasChange gate = %v, want %v", hasChange, tc.wantHasChange)
			}
			if touches != tc.wantTouches {
				t.Errorf("diffTouchesAny gate = %v, want %v", touches, tc.wantTouches)
			}

			// The property that matters: never pay for validation when the plan is empty,
			// always pay for it when it is not.
			if tc.wantPlanEmpty && touches {
				t.Error("plan is empty but the gate would still issue a dry-run save")
			}
			if !tc.wantPlanEmpty && !touches {
				t.Error("plan has changes but the gate would skip validation")
			}
		})
	}
}

// A create has no prior state, so validation must always run.
func TestDatasetDryRunGateAlwaysValidatesOnCreate(t *testing.T) {
	_, _, touches := datasetGateVerdict(t, map[string]string{}, baseDatasetConfig("filter true"))
	if !touches {
		t.Error("gate skipped validation on create; an invalid new dataset would not be caught at plan time")
	}
}

// TestValidationKeysExistInSchema guards the cheap way this gate can rot.
//
// GetChangedKeysPrefix on a key that does not exist in the schema matches nothing and returns
// an empty slice, so a typo or a renamed attribute does not fail -- it silently disables
// validation for that field, forever, with no diagnostic. Assert the lists name real
// attributes.
func TestValidationKeysExistInSchema(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema map[string]*schema.Schema
		keys   []string
	}{
		{"observe_dataset", resourceDataset().Schema, datasetValidationKeys},
		{"observe_log_derived_metric_dataset", resourceLogDerivedMetricDataset().Schema, logDerivedMetricValidationKeys},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.keys) == 0 {
				t.Fatal("validation key list is empty; the dry run would never run")
			}
			for _, k := range tc.keys {
				if _, ok := tc.schema[k]; !ok {
					t.Errorf("validation key %q is not an attribute of %s; "+
						"GetChangedKeysPrefix will never match it and validation for it is silently off", k, tc.name)
				}
			}
		})
	}
}

// TestGateNeverSkipsAValidationRelevantUpdate pins the gate to terraform's own update decision.
//
// The decision lives in vendored SDK code we do not own -- helper/schema/grpc_provider.go,
// PlanResourceChange: `if diff == nil || len(diff.Attributes) == 0 { PlannedState = PriorState }`
// -- so the two cannot be unified into one call. Instead assert the safety property directly
// against the real SDK output:
//
//	gate says "skip validation"  =>  the plan updates nothing under any validated key
//
// That is the property whose violation caused the original bug in reverse (the old HasChange
// gate validated when the plan updated nothing). If a vendored SDK upgrade moves the decision,
// changes what GetChangedKeysPrefix reads, or alters when DiffSuppressFunc prunes, this fails
// rather than drifting silently.
//
// The converse is asserted too, but note it is the weaker direction: it only holds because the
// gate's key list is a subset of the schema, which TestValidationKeysExistInSchema covers.
func TestGateNeverSkipsAValidationRelevantUpdate(t *testing.T) {
	cases := []struct {
		name  string
		state map[string]string
		cfg   map[string]interface{}
	}{
		{
			// Suppressed-only: the case the fix exists for.
			name:  "suppressed_trailing_whitespace",
			state: baseDatasetState("filter true"),
			cfg:   baseDatasetConfig("filter true\n    "),
		},
		{
			name:  "identical",
			state: baseDatasetState("filter true"),
			cfg:   baseDatasetConfig("filter true"),
		},
		{
			// The other stage suppressor; see TestDatasetDryRunGateIgnoresSuppressedDiffs.
			name:  "suppressed_last_stage_alias",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["stage"] = []interface{}{
					map[string]interface{}{"input": "test", "pipeline": "filter true", "alias": "ignored"},
				}
				return c
			}(),
		},
		{
			name:  "real_pipeline_change",
			state: baseDatasetState("filter true"),
			cfg:   baseDatasetConfig("filter false"),
		},
		{
			name:  "real_name_change",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["name"] = "renamed"
				return c
			}(),
		},
		{
			name:  "real_inputs_change",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["inputs"] = map[string]interface{}{"test": "o:::dataset:41000099"}
				return c
			}(),
		},
		{
			// An update confined to a NON-validated attribute. The backend does not need to
			// re-validate the query for it, so the gate should skip -- and this is the case
			// that would regress if someone "simplified" the gate to
			// len(GetChangedKeysPrefix("")) > 0.
			name:  "unvalidated_attribute_only",
			state: baseDatasetState("filter true"),
			cfg: func() map[string]interface{} {
				c := baseDatasetConfig("filter true")
				c["description"] = "changed"
				return c
			}(),
		},
		{
			name:  "create",
			state: map[string]string{},
			cfg:   baseDatasetConfig("filter true"),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := resourceDataset()

			var skip bool
			r.CustomizeDiff = func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
				// The production gate itself, so an edit to it cannot leave this test
				// passing against a stale expression.
				skip = !needsDryRunValidation(d, datasetValidationKeys)
				return nil
			}

			state := &terraform.InstanceState{ID: tc.state["id"], Attributes: tc.state}
			diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(tc.cfg), nil)
			if err != nil {
				t.Fatalf("Diff error: %v", err)
			}

			// What terraform will actually update, restricted to the validated keys. Restricting
			// matters: our own CustomizeDiff may add unrelated keys (resourceDatasetCustomizeDiff
			// calls SetNewComputed("oid")), and those are re-diffed after CustomizeDiff returns,
			// so the final diff is a superset of what the gate saw.
			updatedValidated := []string{}
			if diff != nil {
				for k := range diff.Attributes {
					for _, key := range datasetValidationKeys {
						if k == key || strings.HasPrefix(k, key+".") {
							updatedValidated = append(updatedValidated, k)
							break
						}
					}
				}
			}
			sort.Strings(updatedValidated)

			// THE INVARIANT.
			if skip && len(updatedValidated) > 0 {
				t.Errorf("gate skipped validation, but terraform will update validated attributes %v -- "+
					"an unvalidated change would reach the backend", updatedValidated)
			}
			// And the other direction: never pay for validation when nothing validated changes.
			if !skip && len(updatedValidated) == 0 && state.ID != "" {
				t.Errorf("gate would validate, but terraform updates nothing under %v -- "+
					"this is the wasted-preflight bug returning", datasetValidationKeys)
			}
			t.Logf("skipValidation=%-5v updatesValidatedAttrs=%v", skip, updatedValidated)
		})
	}
}
