package observe

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestDetectOrphanStages(t *testing.T) {
	tests := []struct {
		name    string
		stages  []Stage
		want    []int // nil means no orphans
	}{
		{
			name:   "empty",
			stages: nil,
			want:   nil,
		},
		{
			name: "single stage",
			stages: []Stage{
				{Pipeline: "filter x > 0"},
			},
			want: nil,
		},
		{
			name: "two stages no orphan — second reads from first by default",
			stages: []Stage{
				{Alias: strPtr("a"), Pipeline: "filter x > 0"},
				{Pipeline: "make_col y:x+1"},
			},
			want: nil,
		},
		{
			name: "orphan: first stage bypassed by explicit input on second",
			stages: []Stage{
				{Alias: strPtr("filter_stage"), Pipeline: "filter resourceType ~ \"Firewall\""},
				{Input: strPtr("base_dataset"), Pipeline: "make_col foo:bar"},
			},
			want: []int{0},
		},
		{
			name: "no orphan: second stage explicitly references first stage alias",
			stages: []Stage{
				{Alias: strPtr("prepped"), Pipeline: "filter x > 0"},
				{Input: strPtr("prepped"), Pipeline: "make_col y:x"},
			},
			want: nil,
		},
		{
			name: "no orphan: later stage references alias via @alias in pipeline",
			stages: []Stage{
				{Alias: strPtr("side"), Pipeline: "filter x > 0"},
				{Input: strPtr("base"), Pipeline: "join @side on col=col"},
			},
			want: nil,
		},
		{
			name: "output_stage before end, later stage is its dependency",
			stages: []Stage{
				{OutputStage: true, Input: strPtr("helper"), Pipeline: "join @helper on a=b"},
				{Alias: strPtr("helper"), Pipeline: "filter x > 0"},
			},
			want: nil,
		},
		{
			name: "output_stage mid-list, unreferenced earlier stage is orphan",
			stages: []Stage{
				{Alias: strPtr("orphan"), Pipeline: "filter x < 0"},
				{OutputStage: true, Input: strPtr("base"), Pipeline: "make_col y:1"},
			},
			want: []int{0},
		},
		{
			name: "two orphans in a row",
			stages: []Stage{
				{Alias: strPtr("a"), Pipeline: "filter x > 0"},
				{Alias: strPtr("b"), Pipeline: "filter x > 1"},
				{Input: strPtr("base"), Pipeline: "make_col z:3"},
			},
			want: []int{0, 1},
		},
		{
			name: "anonymous orphan stage bypassed by next stage",
			stages: []Stage{
				{Pipeline: "filter x > 0"},
				{Input: strPtr("base"), Pipeline: "make_col y:1"},
			},
			want: []int{0},
		},
		{
			name: "three stages, middle is orphan",
			stages: []Stage{
				{Alias: strPtr("first"), Pipeline: "filter x > 0"},
				{Alias: strPtr("orphan"), Pipeline: "filter x < 0"},
				{Input: strPtr("first"), Pipeline: "make_col y:1"},
			},
			want: []int{1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectOrphanStages(tc.stages)
			if len(got) != len(tc.want) {
				t.Fatalf("detectOrphanStages() = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("detectOrphanStages()[%d] = %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}
