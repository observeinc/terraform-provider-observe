package observe

import (
	"context"
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
		touches = d.Id() == "" || diffTouchesAny(d, "inputs", "stage", "name")
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
// DiffSuppressFunc hides. stage carries three such suppressors. The observable result on a
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
