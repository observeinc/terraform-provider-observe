package observe

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

// Unit tests for expandEntityTagsFromMap and flattenEntityTagsToMap helper functions

func TestExpandEntityTagsFromMap(t *testing.T) {
	testcases := []struct {
		name     string
		input    map[string]interface{}
		expected []gql.EntityTagMappingInput
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: []gql.EntityTagMappingInput{},
		},
		{
			name: "single value",
			input: map[string]interface{}{
				"environment": "production",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "environment", Values: []string{"production"}},
			},
		},
		{
			name: "multiple values (comma-separated)",
			input: map[string]interface{}{
				"team": "backend,frontend",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
		},
		{
			name: "spaces after commas (normalized)",
			input: map[string]interface{}{
				"team": "backend, frontend, mobile",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend", "mobile"}},
			},
		},
		{
			name: "internal spaces preserved",
			input: map[string]interface{}{
				"description": "Team Alpha,Team Beta",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "description", Values: []string{"Team Alpha", "Team Beta"}},
			},
		},
		{
			name: "leading/trailing spaces trimmed",
			input: map[string]interface{}{
				"team": "  backend  , frontend  ",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
		},
		{
			name: "value with comma (CSV quoted)",
			input: map[string]interface{}{
				"note": "\"Team A, Inc\"",
			},
			expected: []gql.EntityTagMappingInput{
				{Key: "note", Values: []string{"Team A, Inc"}},
			},
		},
		{
			name: "mixed: quoted and unquoted",
			input: map[string]interface{}{
				"tags": "\"Team A, Inc\",backend,frontend",
			},
			expected: []gql.EntityTagMappingInput{
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
			expected: []gql.EntityTagMappingInput{
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
			expected: []gql.EntityTagMappingInput{
				{Key: "description", Values: []string{"Team  Alpha", "Team  Beta"}},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEntityTagsFromMap(tt.input)

			// Sort both slices by Key for consistent comparison (maps are unordered)
			sortEntityTagMappingInputByKey(result)
			expectedCopy := make([]gql.EntityTagMappingInput, len(tt.expected))
			copy(expectedCopy, tt.expected)
			sortEntityTagMappingInputByKey(expectedCopy)

			if diff := cmp.Diff(result, expectedCopy); diff != "" {
				t.Errorf("expandEntityTagsFromMap() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

// TestFlattenEntityTagsToMap tests the flatten function
func TestFlattenEntityTagsToMap(t *testing.T) {
	testcases := []struct {
		name     string
		input    []gql.EntityTagMapping
		expected map[string]interface{}
	}{
		{
			name:     "empty slice",
			input:    []gql.EntityTagMapping{},
			expected: map[string]interface{}{},
		},
		{
			name: "single value",
			input: []gql.EntityTagMapping{
				{Key: "environment", Values: []string{"production"}},
			},
			expected: map[string]interface{}{
				"environment": "production",
			},
		},
		{
			name: "multiple values",
			input: []gql.EntityTagMapping{
				{Key: "team", Values: []string{"backend", "frontend"}},
			},
			expected: map[string]interface{}{
				"team": "backend,frontend",
			},
		},
		{
			name: "internal spaces preserved",
			input: []gql.EntityTagMapping{
				{Key: "description", Values: []string{"Team Alpha", "Team Beta"}},
			},
			expected: map[string]interface{}{
				"description": "Team Alpha,Team Beta",
			},
		},
		{
			name: "CSV quoted values",
			input: []gql.EntityTagMapping{
				{Key: "note", Values: []string{"Team A, Inc"}},
			},
			expected: map[string]interface{}{
				"note": "\"Team A, Inc\"",
			},
		},
		{
			name: "mixed quoted and unquoted",
			input: []gql.EntityTagMapping{
				{Key: "tags", Values: []string{"Team A, Inc", "backend", "frontend"}},
			},
			expected: map[string]interface{}{
				"tags": "\"Team A, Inc\",backend,frontend",
			},
		},
		{
			name: "multiple tags",
			input: []gql.EntityTagMapping{
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
			result := flattenEntityTagsToMap(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("flattenEntityTagsToMap() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

// TestDiffSuppressEntityTagValues tests the diff suppression function
func TestDiffSuppressEntityTagValues(t *testing.T) {
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
			result := diffSuppressEntityTagValues("entity_tags.test", tt.old, tt.new, nil)
			if result != tt.shouldSuppress {
				t.Errorf("diffSuppressEntityTagValues(%q, %q) = %v, want %v",
					tt.old, tt.new, result, tt.shouldSuppress)
			}
		})
	}
}

// Helper function to sort EntityTagMappingInput slices for testing
func sortEntityTagMappingInputByKey(tags []gql.EntityTagMappingInput) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})
}
