package observe

import (
	"bytes"
	"encoding/csv"
	"sort"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const (
	entityTagsDescription       = "Use object_tags instead."
	entityTagsDeprecatedMessage = "Use object_tags instead. entity_tags is deprecated and will be removed in a future release."
)

// objectTagsReader is satisfied by schema.ResourceData (and ResourceDiff).
type objectTagsReader interface {
	GetOk(key string) (interface{}, bool)
}

type rawConfigGetter interface {
	GetRawConfig() cty.Value
}

// objectTagsSchemaFieldOptional returns the canonical object_tags attribute for resources.
func objectTagsSchemaFieldOptional() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeMap,
		Optional:         true,
		DiffSuppressFunc: diffSuppressObjectTagValues,
		Description:      descriptions.Get("common", "schema", "object_tags"),
		ConflictsWith:    []string{"entity_tags"},
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

// entityTagsSchemaFieldOptional returns the deprecated entity_tags attribute for resources.
func entityTagsSchemaFieldOptional() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeMap,
		Optional:         true,
		Description:      entityTagsDescription,
		DiffSuppressFunc: diffSuppressObjectTagValues,
		Deprecated:       entityTagsDeprecatedMessage,
		ConflictsWith:    []string{"object_tags"},
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

// objectTagsSchemaFieldComputed returns object_tags for data sources.
func objectTagsSchemaFieldComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeMap,
		Computed:    true,
		Description: descriptions.Get("common", "schema", "object_tags"),
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

// entityTagsSchemaFieldComputed returns deprecated entity_tags for data sources.
func entityTagsSchemaFieldComputed() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeMap,
		Computed:    true,
		Description: entityTagsDescription,
		Deprecated:  entityTagsDeprecatedMessage,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

// objectTagsInputFromReader reads tags for Create/Update API calls.
// Prefers entity_tags when set in config, matching the SDK v2 optional rename guide.
func objectTagsInputFromReader(r objectTagsReader) []gql.ObjectTagMappingInput {
	field := activeObjectTagsField(r)
	if field == "" {
		return []gql.ObjectTagMappingInput{}
	}
	v, ok := r.GetOk(field)
	if !ok {
		return []gql.ObjectTagMappingInput{}
	}
	return expandObjectTagsFromMap(v.(map[string]interface{}))
}

func tagFieldInRawConfig(cfg cty.Value, field string) bool {
	if cfg.IsNull() || !cfg.Type().IsObjectType() {
		return false
	}
	if !cfg.Type().HasAttribute(field) {
		return false
	}
	return !cfg.GetAttr(field).IsNull()
}

func rawConfigAvailable(cfg cty.Value) bool {
	return !cfg.IsNull() && cfg.Type().IsObjectType()
}

// activeObjectTagsField returns which tag attribute the practitioner is using.
// Config is checked first; state is used during plan refresh when config is unavailable.
func activeObjectTagsField(r objectTagsReader) string {
	if rc, ok := r.(rawConfigGetter); ok {
		cfg := rc.GetRawConfig()
		if tagFieldInRawConfig(cfg, "entity_tags") {
			return "entity_tags"
		}
		if tagFieldInRawConfig(cfg, "object_tags") {
			return "object_tags"
		}
		if rawConfigAvailable(cfg) {
			return ""
		}
	}
	if _, ok := r.GetOk("entity_tags"); ok {
		return "entity_tags"
	}
	if _, ok := r.GetOk("object_tags"); ok {
		return "object_tags"
	}
	return ""
}

// entityTagsInConfigForWarning reports whether entity_tags is in practitioner config.
// schema.Deprecated on TypeMap is not always surfaced during plan; Read uses a state
// fallback when GetRawConfig is unavailable during refresh.
func entityTagsInConfigForWarning(r objectTagsReader) bool {
	return activeObjectTagsField(r) == "entity_tags"
}

// entityTagsDeprecationDiags returns a warning when entity_tags is configured.
func entityTagsDeprecationDiags(r objectTagsReader) diag.Diagnostics {
	if !entityTagsInConfigForWarning(r) {
		return nil
	}
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Argument is deprecated",
		Detail:   entityTagsDeprecatedMessage,
	}}
}

// setObjectTagsFromAPI writes tag values from the API into state and returns
// deprecation warnings for resources using entity_tags.
func setObjectTagsFromAPI(data *schema.ResourceData, tags []gql.ObjectTagMapping, mirrorDeprecatedTag bool) (diag.Diagnostics, error) {
	var err error
	if mirrorDeprecatedTag {
		err = setObjectTagsOnDataSourceData(data, tags)
	} else {
		err = setObjectTagsOnResourceData(data, tags)
	}
	if err != nil {
		return nil, err
	}
	if mirrorDeprecatedTag {
		return nil, nil
	}
	return entityTagsDeprecationDiags(data), nil
}

// setObjectTagsOnResourceData writes API tag values into state on the attribute the
// practitioner configured (entity_tags preferred), matching the SDK v2 optional rename guide.
func setObjectTagsOnResourceData(data *schema.ResourceData, tags []gql.ObjectTagMapping) error {
	flat := flattenObjectTagsToMap(tags)
	field := activeObjectTagsField(data)
	if field == "" {
		field = "object_tags"
	}
	return data.Set(field, flat)
}

// setObjectTagsOnDataSourceData writes both attributes for computed rename compatibility.
func setObjectTagsOnDataSourceData(data *schema.ResourceData, tags []gql.ObjectTagMapping) error {
	flat := flattenObjectTagsToMap(tags)
	if err := data.Set("object_tags", flat); err != nil {
		return err
	}
	return data.Set("entity_tags", flat)
}

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

// expandObjectTagsFromMap converts a Terraform map to GraphQL ObjectTagMappingInput.
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
func expandObjectTagsFromMap(tagsMap map[string]interface{}) []gql.ObjectTagMappingInput {
	result := make([]gql.ObjectTagMappingInput, 0, len(tagsMap))
	for key, valueRaw := range tagsMap {
		result = append(result, gql.ObjectTagMappingInput{
			Key:    key,
			Values: parseCSVValues(valueRaw.(string)),
		})
	}
	return result
}

// flattenObjectTagsToMap converts GraphQL ObjectTagMapping to a Terraform map.
// Multiple values are joined with commas using CSV encoding for proper escaping.
func flattenObjectTagsToMap(tags []gql.ObjectTagMapping) map[string]interface{} {
	result := make(map[string]interface{})
	for _, tag := range tags {
		result[tag.Key] = encodeCSVValues(tag.Values)
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

// diffSuppressObjectTagValues suppresses diffs when object tag values are
// semantically equivalent after normalization (parsing, trimming, and sorting).
//
// Object tag values are treated as unordered sets, following industry standards:
// - Kubernetes labels: alphabetically sorted, order has no semantic meaning
// - AWS tags: unordered key-value pairs
// - Jira/Atlassian labels: always sorted alphabetically, no user control over order
//
// The Observe backend sorts object tag values alphabetically for deterministic output.
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
func diffSuppressObjectTagValues(k, old, new string, d *schema.ResourceData) bool {
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
