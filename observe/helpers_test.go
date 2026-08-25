package observe

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
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

func TestPath(t *testing.T) {
	testcases := []struct {
		Input    string
		HasError bool
	}{
		{
			Input: "",
		},
		{
			Input: "test/path",
		},
		{
			Input: "/test/path",
		},
		{
			Input:    "/test/path?hello",
			HasError: true,
		},
		{
			Input:    "wrong#",
			HasError: true,
		},
	}

	for _, tt := range testcases {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			err := validatePath(tt.Input, nil)

			if tt.HasError && err != nil {
				return
			}

			if tt.HasError && err == nil {
				t.Fatalf("expected error")
			}

			if !tt.HasError && err != nil {
				t.Fatalf("unexpected error")
			}
		})
	}
}

func TestToCamel(t *testing.T) {
	testcases := []struct {
		Input  string
		Expect string
	}{
		{
			Input:  "hello",
			Expect: "Hello",
		},
		{
			Input:  "link_target",
			Expect: "LinkTarget",
		},
		{
			Input:  "not_between_half_open",
			Expect: "NotBetweenHalfOpen",
		},
		{
			Input:  "",
			Expect: "",
		},
	}

	for _, tt := range testcases {
		if result := toCamel(tt.Input); result != tt.Expect {
			t.Fatalf("toCamel failed: expected %s, got %s", tt.Expect, result)
		}

		if result := toSnake(tt.Expect); result != tt.Input {
			t.Fatalf("toSnake failed: expected %s, got %s", tt.Input, result)
		}
	}
}

func TestToCamelLower(t *testing.T) {
	testcases := []struct {
		Input  string
		Expect string
	}{
		{Input: "gauge", Expect: "gauge"},
		{Input: "cumulative_counter", Expect: "cumulativeCounter"},
		{Input: "rate_per_sec", Expect: "ratePerSec"},
		{Input: "exponential_histogram", Expect: "exponentialHistogram"},
		{Input: "", Expect: ""},
	}
	for _, tt := range testcases {
		if result := toCamelLower(tt.Input); result != tt.Expect {
			t.Fatalf("toCamelLower(%q): expected %q, got %q", tt.Input, tt.Expect, result)
		}
	}
}

func TestValidateID(t *testing.T) {
	testcases := []struct {
		input  any
		valid  bool
		expect string
	}{
		{
			input: "123",
			valid: true,
		},
		{
			input: `"123"`,
			valid: false,
		},
		{
			input: "-123",
			valid: false,
		},
		{
			input: "123x",
			valid: false,
		},
		{
			input: 123,
			valid: false,
		},
	}

	for _, tt := range testcases {
		diags := validateID()(tt.input, make(cty.Path, 0))
		if tt.valid {
			if len(diags) != 0 {
				t.Fatalf("should have no validation errors: %v. test: %v", diags, tt)
			}
		} else {
			if len(diags) != 1 {
				t.Fatalf("should have one validation error: %v. test: %v", diags, tt)
			}
		}

	}
}
func TestValidateUID(t *testing.T) {
	testcases := []struct {
		input  any
		valid  bool
		expect string
	}{
		{
			input: "1123",
			valid: true,
		},
		{
			input: "123",
			valid: false, // too small
		},
		{
			input: "10000000",
			valid: false, // too big
		},
		{
			input: `"1123"`,
			valid: true, // allow quoted IDs, see types.StringToUserIdScalar
		},
		{
			input: "-123",
			valid: false,
		},
		{
			input: "123x",
			valid: false,
		},
		{
			input: 123,
			valid: false,
		},
	}

	for _, tt := range testcases {
		diags := validateUID()(tt.input, make(cty.Path, 0))
		if tt.valid {
			if len(diags) != 0 {
				t.Fatalf("should have no validation errors: %v", diags)
			}
		} else {
			if len(diags) != 1 {
				t.Fatalf("should have one validation error: %v", diags)
			}
		}

	}
}

func TestDedentPipeline(t *testing.T) {
	testcases := []struct {
		Name   string
		Input  string
		Expect string
	}{
		{
			Name:   "empty",
			Input:  "",
			Expect: "",
		},
		{
			Name:   "already at column zero",
			Input:  "filter asv = \"idp\"\nstatsby Total: count()",
			Expect: "filter asv = \"idp\"\nstatsby Total: count()",
		},
		{
			Name:   "single indented line",
			Input:  "  filter asv = \"idp\"",
			Expect: "filter asv = \"idp\"",
		},
		{
			Name:   "uniform indent on every line",
			Input:  "  filter asv = \"idp\"\n  filter logGroup ~ \"pipeline\"\n  statsby Total: count()",
			Expect: "filter asv = \"idp\"\nfilter logGroup ~ \"pipeline\"\nstatsby Total: count()",
		},
		{
			Name:   "continuation lines keep their relative indent",
			Input:  "  filter asv = \"idp\"\n  statsby\n      Total: count()",
			Expect: "filter asv = \"idp\"\nstatsby\n    Total: count()",
		},
		{
			Name:   "continuation lines when the first verb is already at column zero",
			Input:  "filter asv = \"idp\"\nstatsby\n    Total: count()",
			Expect: "filter asv = \"idp\"\nstatsby\n    Total: count()",
		},
		{
			Name:   "blank lines are not padded out to the common indent",
			Input:  "    filter asv = \"idp\"\n  \n    statsby Total: count()",
			Expect: "filter asv = \"idp\"\n\nstatsby Total: count()",
		},
		{
			Name:   "trailing newline preserved",
			Input:  "  filter asv = \"idp\"\n",
			Expect: "filter asv = \"idp\"\n",
		},
		{
			Name:   "tab indent",
			Input:  "\tfilter asv = \"idp\"\n\tstatsby Total: count()",
			Expect: "filter asv = \"idp\"\nstatsby Total: count()",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			if result := dedentPipeline(tt.Input); result != tt.Expect {
				t.Fatalf("dedentPipeline(%q) = %q, expected %q", tt.Input, result, tt.Expect)
			}
		})
	}
}

// TestDiffSuppressStageQueryInput_DatasetIdOIDVsBare exercises the fix for the
// perpetual diff described in task-364 §2e: a stage's top-level input[].datasetId
// is stored and returned by the backend as a bare numeric id, even when config
// supplies a full dataset OID. Without normalization, the two forms compare as
// unequal strings and the diff never converges.
func TestDiffSuppressStageQueryInput_DatasetIdOIDVsBare(t *testing.T) {
	testcases := []struct {
		name     string
		prv      string
		nxt      string
		suppress bool
	}{
		{
			name:     "full OID in config, bare id in state -> suppressed",
			prv:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			nxt:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"o:::dataset:41036871"}]}]`,
			suppress: true,
		},
		{
			name:     "bare id in config, full OID in state -> suppressed",
			prv:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"o:::dataset:41036871"}]}]`,
			nxt:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			suppress: true,
		},
		{
			name:     "both bare and equal -> suppressed",
			prv:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			nxt:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			suppress: true,
		},
		{
			name:     "different dataset ids -> not suppressed",
			prv:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			nxt:      `[{"pipeline":"","input":[{"inputName":"a","inputRole":"Data","datasetId":"o:::dataset:99999999"}]}]`,
			suppress: false,
		},
		{
			name:     "pipeline text changed -> not suppressed",
			prv:      `[{"pipeline":"filter a","input":[{"inputName":"a","inputRole":"Data","datasetId":"41036871"}]}]`,
			nxt:      `[{"pipeline":"filter b","input":[{"inputName":"a","inputRole":"Data","datasetId":"o:::dataset:41036871"}]}]`,
			suppress: false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			got := diffSuppressStageQueryInput("stages", tt.prv, tt.nxt, nil)
			if got != tt.suppress {
				t.Errorf("diffSuppressStageQueryInput() = %v, want %v", got, tt.suppress)
			}
		})
	}
}

// newMultilineErrorRegexp creates a regexp that matches the given string,
// allowing for any whitespace (including newlines) anywhere a space is present
// in the input. The Terraform provider test framework executes the Terraform
// CLI, which wraps errors returned from providers at a particular column width.
// This makes test steps that use ExpectError especially brittle with longer
// error messages, which may wrap at a different word if the existing error
// message is prefixed with additional text.
func newMultilineErrorRegexp(s string) *regexp.Regexp {
	s = strings.ReplaceAll(s, " ", `\s`)
	return regexp.MustCompile(s)
}
