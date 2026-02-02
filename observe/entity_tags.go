package observe

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

// parseCSVValues parses a CSV string and trims whitespace from each value.
// Does NOT sort - the backend will sort values, so client-side sorting is redundant.
func parseCSVValues(csvStr string) []string {
	// Parse as CSV to handle comma-separated values and proper escaping
	values, err := csv.NewReader(strings.NewReader(csvStr)).Read()
	if err != nil {
		// If CSV parsing fails, treat as single value
		values = []string{csvStr}
	}

	// Trim leading/trailing whitespace from each value (internal spaces preserved)
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}

	return values
}

// expandEntityTagsFromMap converts a Terraform map to GraphQL EntityTagMappingInput.
// Values are parsed as CSV to support multiple values per key.
//
// Examples:
//
//	"team" = "backend,frontend"           → ["backend", "frontend"]
//	"team" = "backend, frontend"          → ["backend", "frontend"] (spaces trimmed)
//	"desc" = "Team Alpha,Team Beta"       → ["Team Alpha", "Team Beta"] (internal spaces preserved)
//	"note" = "\"Value with, comma\""      → ["Value with, comma"] (CSV escaping)
//
// Note: Backend sorts values alphabetically. We don't sort here to avoid redundant work.
// DiffSuppressFunc handles state drift by normalizing both config and state values.
func expandEntityTagsFromMap(tagsMap map[string]interface{}) []gql.EntityTagMappingInput {
	result := make([]gql.EntityTagMappingInput, 0, len(tagsMap))
	for key, valueRaw := range tagsMap {
		result = append(result, gql.EntityTagMappingInput{
			Key:    key,
			Values: parseCSVValues(valueRaw.(string)),
		})
	}
	return result
}

// flattenEntityTagsToMap converts GraphQL EntityTagMapping to a Terraform map.
// Multiple values are joined with commas using CSV encoding for proper escaping.
//
// Uses reflection to handle all EntityTagMapping types generically, avoiding
// the need for a type switch with duplicate code for each resource type.
func flattenEntityTagsToMap(tags interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Use reflection to iterate over any slice of EntityTagMapping structs
	v := reflect.ValueOf(tags)
	if v.Kind() != reflect.Slice {
		return result
	}

	for i := 0; i < v.Len(); i++ {
		tag := v.Index(i)
		// All EntityTagMapping types have Key and Values fields
		key := tag.FieldByName("Key").String()
		values := tag.FieldByName("Values").Interface().([]string)
		result[key] = encodeCSVValues(values)
	}

	return result
}

// encodeCSVValues encodes a slice of strings as a CSV string.
// Uses CSV writer to properly escape values containing commas.
func encodeCSVValues(values []string) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(values)
	w.Flush()
	return strings.TrimSuffix(buf.String(), "\n")
}

// diffSuppressEntityTagValues suppresses diffs when entity tag values are
// semantically equivalent after normalization (parsing, trimming, and sorting).
//
// Entity tag values are treated as unordered sets, following industry standards:
// - Kubernetes labels: alphabetically sorted, order has no semantic meaning
// - AWS tags: unordered key-value pairs
// - Jira/Atlassian labels: always sorted alphabetically, no user control over order
//
// The Observe backend sorts entity tag values alphabetically for deterministic output.
// This function normalizes both old (from state) and new (from config) values by
// sorting them alphabetically to prevent false diffs when users specify values in
// a different order.
//
// This is the ONLY place where we need to sort values. Examples:
//   - User writes: "high,critical" in config
//   - Backend returns: "critical,high" in state (backend sorts alphabetically)
//   - After normalization: both become ["critical", "high"] → no diff
//
// Additional normalization examples:
//   - "production,staging" == "staging,production" (different order)
//   - "a, b, c" == "a,b,c" (extra whitespace trimmed)
//   - "z,a,m" == "a,m,z" (backend will sort both)
func diffSuppressEntityTagValues(k, old, new string, d *schema.ResourceData) bool {
	// Parse and normalize both values
	oldValues := parseCSVValues(old)
	newValues := parseCSVValues(new)

	// Sort for comparison (backend always returns sorted values)
	sort.Strings(oldValues)
	sort.Strings(newValues)

	// Compare lengths first (fast path)
	if len(oldValues) != len(newValues) {
		return false
	}

	// Compare each value
	for i := range oldValues {
		if oldValues[i] != newValues[i] {
			return false
		}
	}
	return true
}
