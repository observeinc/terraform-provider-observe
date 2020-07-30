package observe

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlags(t *testing.T) {
	testcases := []struct {
		Input    string
		Expected map[string]bool
		HasError bool
	}{
		{
			Input:    "",
			Expected: map[string]bool{},
		},
		{
			Input: "!hello",
			Expected: map[string]bool{
				"hello": false,
			},
		},
		{
			Input: "a,b,!c",
			Expected: map[string]bool{
				"a": true,
				"b": true,
				"c": false,
			},
		},
		{
			// technically allowed, last flag wins
			Input: "!hello,hello",
			Expected: map[string]bool{
				"hello": true,
			},
		},
		{
			// no caps
			Input:    "AA",
			HasError: true,
		},
		{
			// no empty items
			Input:    ",a",
			HasError: true,
		},
		{
			// no leading digit
			Input:    "12",
			HasError: true,
		},
	}

	for _, tt := range testcases {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			result, err := convertFlags(tt.Input)

			if tt.HasError && err != nil {
				return
			}

			if tt.HasError && err == nil {
				t.Fatalf("expected error")
			}

			if !tt.HasError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if s := cmp.Diff(result, tt.Expected); s != "" {
				t.Fatalf("unexpected result: %s", s)
			}
		})
	}
}
