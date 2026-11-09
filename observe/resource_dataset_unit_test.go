package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

// TestFlattenAndSetQueryStageInput tests that flattenAndSetQuery correctly
// preserves or omits the stage.input field across all cases: single-input first
// stage, aliased-chain stages (including the redundant-input perpetual-diff bug),
// and non-redundant explicit inputs. See TestAccObserveDatasetRedundantAliasInputNoPerpetualDiff
// for the end-to-end acceptance-test regression guard for the bug itself.
func TestFlattenAndSetQueryStageInput(t *testing.T) {
	const (
		wsOID  = "o:::workspace:41000215"
		ds1OID = "o:::dataset:41000001"
		ds2OID = "o:::dataset:41000002"
		ds1ID  = "41000001"
		ds2ID  = "41000002"
	)

	ptr := func(s string) *string { return &s }

	cases := []struct {
		name         string
		configInputs map[string]interface{}
		configStages []interface{}
		gqlStages    []gql.StageQuery
		outputStage  string
		wantInputs   []string // expected stage[i].input in state after flattenAndSetQuery
	}{
		{
			// Stage 0 with a single external input and no explicit input in
			// config. flattenQuery omits it (defaulted from the sole input);
			// the fallback reads config = "" → state stays "".
			name:         "stage0_single_input_no_config_input",
			configInputs: map[string]interface{}{"ds": ds1OID},
			configStages: []interface{}{
				map[string]interface{}{"pipeline": "filter true"},
			},
			gqlStages: []gql.StageQuery{
				{
					Id:       ptr("stage-0"),
					Pipeline: "filter true",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds", DatasetId: ptr(ds1ID)},
					},
				},
			},
			outputStage: "stage-0",
			wantInputs:  []string{""},
		},
		{
			// Stage 0 with a single external input and an explicit input = "ds"
			// in config. flattenQuery omits it (defaulted), but the fallback
			// preserves the config value → state gets "ds".
			name:         "stage0_single_input_explicit_config_input",
			configInputs: map[string]interface{}{"ds": ds1OID},
			configStages: []interface{}{
				map[string]interface{}{"input": "ds", "pipeline": "filter true"},
			},
			gqlStages: []gql.StageQuery{
				{
					Id:       ptr("stage-0"),
					Pipeline: "filter true",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds", DatasetId: ptr(ds1ID)},
					},
				},
			},
			outputStage: "stage-0",
			wantInputs:  []string{"ds"},
		},
		{
			// Stage 1 chains implicitly from aliased stage 0; config has no
			// explicit input on stage 1. flattenQuery omits it (alias chain);
			// fallback reads config = "" → state stays "".
			name:         "aliased_chain_no_config_input",
			configInputs: map[string]interface{}{"ds": ds1OID},
			configStages: []interface{}{
				map[string]interface{}{"alias": "base_stage", "pipeline": "filter true"},
				map[string]interface{}{"pipeline": "filter false"},
			},
			gqlStages: []gql.StageQuery{
				{
					Id:       ptr("stage-0"),
					Pipeline: "filter true",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds", DatasetId: ptr(ds1ID)},
					},
				},
				{
					Id:       ptr("stage-1"),
					Pipeline: "filter false",
					Input: []gql.StageQueryInputInputDefinition{
						// InputName matches the alias; StageId ties it back to stage-0
						{InputName: "base_stage", StageId: ptr("stage-0")},
					},
				},
			},
			outputStage: "stage-1",
			wantInputs:  []string{"", ""},
		},
		{
			// THE BUG FIX: stage 1 chains from aliased stage 0 but config sets
			// input = "base_stage" explicitly (the redundant form). Before the
			// fix, flattenAndSetQuery cleared it → state got ""; after the fix,
			// the config value is preserved → state gets "base_stage".
			name:         "aliased_chain_explicit_redundant_config_input",
			configInputs: map[string]interface{}{"ds": ds1OID},
			configStages: []interface{}{
				map[string]interface{}{"alias": "base_stage", "pipeline": "filter true"},
				map[string]interface{}{"input": "base_stage", "pipeline": "filter false"},
			},
			gqlStages: []gql.StageQuery{
				{
					Id:       ptr("stage-0"),
					Pipeline: "filter true",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds", DatasetId: ptr(ds1ID)},
					},
				},
				{
					Id:       ptr("stage-1"),
					Pipeline: "filter false",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "base_stage", StageId: ptr("stage-0")},
					},
				},
			},
			outputStage: "stage-1",
			wantInputs:  []string{"", "base_stage"},
		},
		{
			// Stage 1 has a non-redundant explicit input pointing to a second
			// external dataset. flattenQuery's default branch sets Input, so the
			// fix's fallback path is never reached. No behavioral change expected.
			name:         "non_redundant_explicit_input_to_second_dataset",
			configInputs: map[string]interface{}{"ds1": ds1OID, "ds2": ds2OID},
			configStages: []interface{}{
				map[string]interface{}{"pipeline": "filter true"},
				map[string]interface{}{"input": "ds2", "pipeline": "filter false"},
			},
			gqlStages: []gql.StageQuery{
				{
					Id:       ptr("stage-0"),
					Pipeline: "filter true",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds1", DatasetId: ptr(ds1ID)},
					},
				},
				{
					Id:       ptr("stage-1"),
					Pipeline: "filter false",
					Input: []gql.StageQueryInputInputDefinition{
						{InputName: "ds2", DatasetId: ptr(ds2ID)},
					},
				},
			},
			outputStage: "stage-1",
			wantInputs:  []string{"ds1", "ds2"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{
				"workspace": wsOID,
				"name":      "test-dataset",
				"inputs":    tc.configInputs,
				"stage":     tc.configStages,
			}
			d := schema.TestResourceDataRaw(t, resourceDataset().Schema, raw)

			if _, err := flattenAndSetQuery(d, tc.gqlStages, tc.outputStage, false); err != nil {
				t.Fatalf("flattenAndSetQuery error: %v", err)
			}

			stages := d.Get("stage").([]interface{})
			if len(stages) != len(tc.wantInputs) {
				t.Fatalf("got %d stages, want %d", len(stages), len(tc.wantInputs))
			}
			for i, want := range tc.wantInputs {
				s, ok := stages[i].(map[string]interface{})
				if !ok {
					t.Fatalf("stage[%d] is not a map", i)
				}
				got, _ := s["input"].(string)
				if got != want {
					t.Errorf("stage[%d].input = %q, want %q", i, got, want)
				}
			}
		})
	}
}

// TestFlattenAndSetQueryOutputStage covers the OB-67131 / OB-44450 perpetual diff.
//
// newQuery sends the last stage as the output stage whether or not output_stage is
// declared there, so the mapping is genuinely not invertible: given outputStage == the
// last stage's id, the config could have declared it or omitted it. flattenQuery's
// positional guard (i < len(gqlStages)-1) resolves that ambiguity by reporting false,
// which is right for the common omitted case and lossy for the explicit one. Config then
// said true, refreshed state said false, and every plan showed the change forever.
//
// Because the read cannot be made correct, the fix suppresses the diff in the schema.
// Declaring output_stage on the last stage is semantically a no-op -- that stage is
// already the output stage -- so nothing is lost. This test pins both halves: the read
// stays lossy (deliberately), and the suppression covers exactly the last stage.
// See TestAccObserveDatasetLastStageOutputStageNoPerpetualDiff for the end-to-end guard.
func TestFlattenAndSetQueryOutputStage(t *testing.T) {
	const (
		wsOID  = "o:::workspace:41000215"
		ds1OID = "o:::dataset:41000001"
		ds1ID  = "41000001"
	)

	// Two stages as the backend returns them: stage-1 chains from stage-0.
	gqlStages := []gql.StageQuery{
		{
			Id:       strPtr("stage-0"),
			Pipeline: "filter true",
			Input: []gql.StageQueryInputInputDefinition{
				{InputName: "ds", DatasetId: strPtr(ds1ID)},
			},
		},
		{
			Id:       strPtr("stage-1"),
			Pipeline: "filter false",
			Input: []gql.StageQueryInputInputDefinition{
				{InputName: "stage-0", StageId: strPtr("stage-0")},
			},
		},
	}

	cases := []struct {
		name         string
		configStages []interface{}
		outputStage  string
		// expected stage[i].output_stage in state after flattenAndSetQuery
		wantOutputStages []bool
		// expected DiffSuppressFunc verdict for each stage[i].output_stage key
		wantSuppressed []bool
	}{
		{
			// THE BUG: config declares output_stage on the last stage, state comes
			// back false. Suppression on that index is what keeps the plan empty.
			name: "declared_on_last_stage_is_lossy_but_suppressed",
			configStages: []interface{}{
				map[string]interface{}{"pipeline": "filter true"},
				map[string]interface{}{"pipeline": "filter false", "output_stage": true},
			},
			outputStage:      "stage-1",
			wantOutputStages: []bool{false, false},
			wantSuppressed:   []bool{false, true},
		},
		{
			// The common case the positional guard exists to protect: nobody declares
			// an output stage, so state must stay false or we would invent a diff.
			name: "omitted_stays_false",
			configStages: []interface{}{
				map[string]interface{}{"pipeline": "filter true"},
				map[string]interface{}{"pipeline": "filter false"},
			},
			outputStage:      "stage-1",
			wantOutputStages: []bool{false, false},
			wantSuppressed:   []bool{false, true},
		},
		{
			// Declared on an earlier stage: unambiguous, so the read round-trips it and
			// the suppression must NOT cover it -- a real change there still diffs.
			name: "declared_on_earlier_stage_round_trips_and_is_not_suppressed",
			configStages: []interface{}{
				map[string]interface{}{"pipeline": "filter true", "output_stage": true},
				map[string]interface{}{"pipeline": "filter false"},
			},
			outputStage:      "stage-0",
			wantOutputStages: []bool{true, false},
			wantSuppressed:   []bool{false, true},
		},
	}

	suppress := resourceDataset().Schema["stage"].Elem.(*schema.Resource).Schema["output_stage"].DiffSuppressFunc
	if suppress == nil {
		t.Fatal("stage.output_stage has no DiffSuppressFunc; the OB-67131 perpetual diff is back")
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{
				"workspace": wsOID,
				"name":      "test-dataset",
				"inputs":    map[string]interface{}{"ds": ds1OID},
				"stage":     tc.configStages,
			}
			d := schema.TestResourceDataRaw(t, resourceDataset().Schema, raw)

			if _, err := flattenAndSetQuery(d, gqlStages, tc.outputStage, false); err != nil {
				t.Fatalf("flattenAndSetQuery error: %v", err)
			}

			stages := d.Get("stage").([]interface{})
			if len(stages) != len(tc.wantOutputStages) {
				t.Fatalf("got %d stages, want %d", len(stages), len(tc.wantOutputStages))
			}
			for i, want := range tc.wantOutputStages {
				s, ok := stages[i].(map[string]interface{})
				if !ok {
					t.Fatalf("stage[%d] is not a map", i)
				}
				got, _ := s["output_stage"].(bool)
				if got != want {
					t.Errorf("stage[%d].output_stage in state = %v, want %v", i, got, want)
				}
			}

			for i, want := range tc.wantSuppressed {
				key := fmt.Sprintf("stage.%d.output_stage", i)
				// old = what the read stored, new = what config asks for.
				if got := suppress(key, "false", "true", d); got != want {
					t.Errorf("DiffSuppressFunc(%q) = %v, want %v", key, got, want)
				}
			}
		})
	}
}
