package observe

import (
	"regexp"
	"strings"
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
