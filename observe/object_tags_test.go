package observe

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

// Unit tests for expandObjectTagsFromMap and flattenObjectTagsToMap helper functions

func TestExpandObjectTagsFromMap(t *testing.T) {
	testcases := []struct {
		name     string
		input    map[string]interface{}
		expected []gql.ObjectTagMappingInput
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: []gql.ObjectTagMappingInput{},
		},
		{
			name: "single value",
			input: map[string]interface{}{
				"environment": "production",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "environment", Values: []string{"production"}},
			},
		},
		{
			name: "multiple values (comma-separated)",
			input: map[string]interface{}{
				"team": "backend,frontend",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
		},
		{
			name: "spaces after commas (normalized)",
			input: map[string]interface{}{
				"team": "backend, frontend, mobile",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend", "mobile"}},
			},
		},
		{
			name: "internal spaces preserved",
			input: map[string]interface{}{
				"description": "Team Alpha,Team Beta",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "description", Values: []string{"Team Alpha", "Team Beta"}},
			},
		},
		{
			name: "leading/trailing spaces trimmed",
			input: map[string]interface{}{
				"team": "  backend  , frontend  ",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
		},
		{
			name: "value with comma (CSV quoted)",
			input: map[string]interface{}{
				"note": "\"Team A, Inc\"",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "note", Values: []string{"Team A, Inc"}},
			},
		},
		{
			name: "mixed: quoted and unquoted",
			input: map[string]interface{}{
				"tags": "\"Team A, Inc\",backend,frontend",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "tags", Values: []string{"Team A, Inc", "backend", "frontend"}},
			},
		},
		{
			name: "multiple tags",
			input: map[string]interface{}{
				"environment": "production",
				"team":        "backend,frontend",
				"region":      "us-west-2",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "environment", Values: []string{"production"}},
				{Key: "team", Values: []string{"backend", "frontend"}},
				{Key: "region", Values: []string{"us-west-2"}},
			},
		},
		{
			name: "internal spaces with multiple spaces preserved",
			input: map[string]interface{}{
				"description": "Team  Alpha,Team  Beta",
			},
			expected: []gql.ObjectTagMappingInput{
				{Key: "description", Values: []string{"Team  Alpha", "Team  Beta"}},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := expandObjectTagsFromMap(tt.input)

			// Sort both slices by Key for consistent comparison (maps are unordered)
			sortObjectTagMappingInputByKey(result)
			expectedCopy := make([]gql.ObjectTagMappingInput, len(tt.expected))
			copy(expectedCopy, tt.expected)
			sortObjectTagMappingInputByKey(expectedCopy)

			if diff := cmp.Diff(result, expectedCopy); diff != "" {
				t.Errorf("expandObjectTagsFromMap() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

// TestFlattenObjectTagsToMap tests the flatten function
func TestFlattenObjectTagsToMap(t *testing.T) {
	testcases := []struct {
		name     string
		input    []gql.ObjectTagMapping
		expected map[string]interface{}
	}{
		{
			name:     "empty slice",
			input:    []gql.ObjectTagMapping{},
			expected: map[string]interface{}{},
		},
		{
			name: "single value",
			input: []gql.ObjectTagMapping{
				{Key: "environment", Values: []string{"production"}},
			},
			expected: map[string]interface{}{
				"environment": "production",
			},
		},
		{
			name: "multiple values",
			input: []gql.ObjectTagMapping{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
			expected: map[string]interface{}{
				"team": "backend,frontend",
			},
		},
		{
			name: "internal spaces preserved",
			input: []gql.ObjectTagMapping{
				{Key: "description", Values: []string{"Team Alpha", "Team Beta"}},
			},
			expected: map[string]interface{}{
				"description": "Team Alpha,Team Beta",
			},
		},
		{
			name: "CSV quoted values",
			input: []gql.ObjectTagMapping{
				{Key: "note", Values: []string{"Team A, Inc"}},
			},
			expected: map[string]interface{}{
				"note": "\"Team A, Inc\"",
			},
		},
		{
			name: "mixed quoted and unquoted",
			input: []gql.ObjectTagMapping{
				{Key: "tags", Values: []string{"Team A, Inc", "backend", "frontend"}},
			},
			expected: map[string]interface{}{
				"tags": "\"Team A, Inc\",backend,frontend",
			},
		},
		{
			name: "multiple tags",
			input: []gql.ObjectTagMapping{
				{Key: "environment", Values: []string{"production"}},
				{Key: "team", Values: []string{"backend", "frontend"}},
				{Key: "region", Values: []string{"us-west-2"}},
			},
			expected: map[string]interface{}{
				"environment": "production",
				"team":        "backend,frontend",
				"region":      "us-west-2",
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenObjectTagsToMap(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("flattenObjectTagsToMap() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

// TestDiffSuppressObjectTagValues tests the diff suppression function
func TestDiffSuppressObjectTagValues(t *testing.T) {
	testcases := []struct {
		name           string
		old            string
		new            string
		shouldSuppress bool
	}{
		// Should suppress (semantically equivalent)
		{name: "identical values", old: "production", new: "production", shouldSuppress: true},
		{name: "different order", old: "backend,frontend", new: "frontend,backend", shouldSuppress: true},
		{name: "different spacing", old: "backend, frontend", new: "backend,frontend", shouldSuppress: true},
		{name: "extra spaces", old: "  backend  ,  frontend  ", new: "backend,frontend", shouldSuppress: true},
		{name: "complex reordering", old: "critical, high, medium", new: "medium,high,critical", shouldSuppress: true},
		{name: "both empty", old: "", new: "", shouldSuppress: true},
		{name: "internal spaces preserved", old: "Team Alpha,Team Beta", new: "Team Beta,Team Alpha", shouldSuppress: true},
		{name: "CSV quoted values", old: "\"Team A, Inc\",backend", new: "backend,\"Team A, Inc\"", shouldSuppress: true},

		// Should NOT suppress (semantically different)
		{name: "different values", old: "backend,frontend", new: "backend,mobile", shouldSuppress: false},
		{name: "different length", old: "backend,frontend", new: "backend,frontend,mobile", shouldSuppress: false},
		{name: "completely different", old: "production", new: "staging", shouldSuppress: false},
		{name: "empty vs non-empty", old: "", new: "production", shouldSuppress: false},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := diffSuppressObjectTagValues("object_tags.test", tt.old, tt.new, nil)
			if result != tt.shouldSuppress {
				t.Errorf("diffSuppressObjectTagValues(%q, %q) = %v, want %v",
					tt.old, tt.new, result, tt.shouldSuppress)
			}
		})
	}
}

type stubObjectTagsReader map[string]interface{}

func (s stubObjectTagsReader) GetOk(key string) (interface{}, bool) {
	v, ok := s[key]
	return v, ok
}

type stubObjectTagsReaderWithConfig struct {
	stubObjectTagsReader
	rawConfig cty.Value
}

func (s stubObjectTagsReaderWithConfig) GetRawConfig() cty.Value {
	return s.rawConfig
}

func TestObjectTagsInputFromReader(t *testing.T) {
	tags := map[string]interface{}{"team": "backend,frontend"}

	t.Run("prefers entity_tags", func(t *testing.T) {
		got := objectTagsInputFromReader(stubObjectTagsReader{
			"object_tags": map[string]interface{}{"ignored": "value"},
			"entity_tags": tags,
		})
		want := []gql.ObjectTagMappingInput{{Key: "team", Values: []string{"backend", "frontend"}}}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("objectTagsInputFromReader() mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("falls back to object_tags", func(t *testing.T) {
		got := objectTagsInputFromReader(stubObjectTagsReader{"object_tags": tags})
		want := []gql.ObjectTagMappingInput{{Key: "team", Values: []string{"backend", "frontend"}}}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("objectTagsInputFromReader() mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("prefers object_tags in config over entity_tags in state", func(t *testing.T) {
		got := objectTagsInputFromReader(stubObjectTagsReaderWithConfig{
			stubObjectTagsReader: stubObjectTagsReader{
				"entity_tags": map[string]interface{}{"ignored": "value"},
				"object_tags": tags,
			},
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"object_tags": cty.MapVal(map[string]cty.Value{
					"team": cty.StringVal("backend,frontend"),
				}),
			}),
		})
		want := []gql.ObjectTagMappingInput{{Key: "team", Values: []string{"backend", "frontend"}}}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("objectTagsInputFromReader() mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("empty when neither set", func(t *testing.T) {
		got := objectTagsInputFromReader(stubObjectTagsReader{})
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})
}

func TestEntityTagsDeprecationDiags(t *testing.T) {
	t.Run("warns when entity_tags only in state", func(t *testing.T) {
		diags := entityTagsDeprecationDiags(stubObjectTagsReader{
			"entity_tags": map[string]interface{}{"env": "prod"},
		})
		if len(diags) != 1 || diags[0].Detail != entityTagsDeprecatedMessage {
			t.Fatalf("expected deprecation warning, got %#v", diags)
		}
	})

	t.Run("no warning when object_tags only in state", func(t *testing.T) {
		diags := entityTagsDeprecationDiags(stubObjectTagsReader{
			"object_tags": map[string]interface{}{"env": "prod"},
		})
		if len(diags) != 0 {
			t.Fatalf("expected no warning, got %#v", diags)
		}
	})

	t.Run("no warning when neither set", func(t *testing.T) {
		diags := entityTagsDeprecationDiags(stubObjectTagsReader{})
		if len(diags) != 0 {
			t.Fatalf("expected no warning, got %#v", diags)
		}
	})
}

func TestEntityTagsSchemaValidateDeprecation(t *testing.T) {
	res := Provider().ResourcesMap["observe_dataset"]
	cfgVal := cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("test"),
		"entity_tags": cty.MapVal(map[string]cty.Value{
			"env": cty.StringVal("prod"),
		}),
	})
	config := terraform.NewResourceConfigShimmed(cfgVal, res.CoreConfigSchema())
	diags := res.Validate(config)

	var found bool
	for _, d := range diags {
		if d.Severity == diag.Warning && d.Detail == entityTagsDeprecatedMessage {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected entity_tags deprecation warning in validate diags, got %#v", diags)
	}
}

// Helper function to sort ObjectTagMappingInput slices for testing
func sortObjectTagMappingInputByKey(tags []gql.ObjectTagMappingInput) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})
}
