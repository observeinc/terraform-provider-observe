package observe

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// escapeHCLTemplateMarkers escapes HCL template interpolation and directive
// markers so that literal customer text containing "${" or "%{" survives
// rendering through `terraform show` (which outputs heredocs that interpret
// these markers).
//
// The two replacements are order-independent: neither can produce the other's
// trigger sequence, and applying them to already-escaped input (e.g. "$${")
// yields "$$${" which round-trips correctly through HCL.
func escapeHCLTemplateMarkers(s string) string {
	s = strings.ReplaceAll(s, "${", "$${")
	s = strings.ReplaceAll(s, "%{", "%%{")
	return s
}

// escapeExportedStrings walks the schema of a data source and escapes HCL
// template markers in every TypeString value currently held in data. It also
// recurses into TypeList/TypeSet elements backed by a nested *schema.Resource,
// and escapes TypeString elements in primitive string lists.
//
// Only attributes whose values actually contain markers are written back via
// data.Set — unchanged attributes are left untouched so the exported state
// stays byte-identical for objects that happen to contain no markers.
func escapeExportedStrings(data *schema.ResourceData, s map[string]*schema.Schema) error {
	for key, attr := range s {
		switch attr.Type {
		case schema.TypeString:
			raw := data.Get(key)
			if raw == nil {
				continue
			}
			v := raw.(string)
			if v == "" {
				continue
			}
			escaped := escapeHCLTemplateMarkers(v)
			if escaped != v {
				if err := data.Set(key, escaped); err != nil {
					return err
				}
			}

		case schema.TypeList, schema.TypeSet:
			if attr.Elem == nil {
				continue
			}
			switch elem := attr.Elem.(type) {
			case *schema.Resource:
				// Nested block — recurse into each element.
				if changed, err := escapeNestedBlock(data.Get(key), elem.Schema); err != nil {
					return err
				} else if changed {
					if err := data.Set(key, data.Get(key)); err != nil {
						return err
					}
				}
			case *schema.Schema:
				// Primitive string list/set — escape each element.
				if elem.Type != schema.TypeString {
					continue
				}
				if changed, err := escapeStringList(data, key); err != nil {
					return err
				} else if !changed {
					continue
				}
			}
		}
	}
	return nil
}

// escapeNestedBlock walks through a list/set of maps (the SDK representation
// of repeated nested blocks) and escapes TypeString leaves. Returns true if
// any value was modified.
func escapeNestedBlock(raw interface{}, s map[string]*schema.Schema) (bool, error) {
	items, ok := toSlice(raw)
	if !ok || len(items) == 0 {
		return false, nil
	}

	anyChanged := false
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for key, attr := range s {
			switch attr.Type {
			case schema.TypeString:
				v, ok := m[key].(string)
				if !ok || v == "" {
					continue
				}
				escaped := escapeHCLTemplateMarkers(v)
				if escaped != v {
					m[key] = escaped
					anyChanged = true
				}

			case schema.TypeList, schema.TypeSet:
				if attr.Elem == nil {
					continue
				}
				switch elem := attr.Elem.(type) {
				case *schema.Resource:
					changed, err := escapeNestedBlock(m[key], elem.Schema)
					if err != nil {
						return false, err
					}
					if changed {
						anyChanged = true
					}
				case *schema.Schema:
					if elem.Type != schema.TypeString {
						continue
					}
					changed, err := escapeStringSlice(m, key)
					if err != nil {
						return false, err
					}
					if changed {
						anyChanged = true
					}
				}
			}
		}
	}
	return anyChanged, nil
}

// escapeStringList escapes template markers in a top-level TypeList or TypeSet
// of strings accessed via ResourceData. Returns true if any element changed.
func escapeStringList(data *schema.ResourceData, key string) (bool, error) {
	raw := data.Get(key)
	items, ok := toSlice(raw)
	if !ok || len(items) == 0 {
		return false, nil
	}

	anyChanged := false
	result := make([]interface{}, len(items))
	for i, item := range items {
		v, ok := item.(string)
		if !ok {
			result[i] = item
			continue
		}
		escaped := escapeHCLTemplateMarkers(v)
		if escaped != v {
			anyChanged = true
		}
		result[i] = escaped
	}
	if !anyChanged {
		return false, nil
	}
	return true, data.Set(key, result)
}

// escapeStringSlice escapes template markers in a nested string list stored in
// a map (the SDK representation inside a nested block). Returns true if any
// element changed.
func escapeStringSlice(m map[string]interface{}, key string) (bool, error) {
	raw, exists := m[key]
	if !exists {
		return false, nil
	}
	items, ok := toSlice(raw)
	if !ok || len(items) == 0 {
		return false, nil
	}

	anyChanged := false
	result := make([]interface{}, len(items))
	for i, item := range items {
		v, ok := item.(string)
		if !ok {
			result[i] = item
			continue
		}
		escaped := escapeHCLTemplateMarkers(v)
		if escaped != v {
			anyChanged = true
		}
		result[i] = escaped
	}
	if anyChanged {
		m[key] = result
	}
	return anyChanged, nil
}

// toSlice normalizes the SDK's list/set representation to []interface{}.
func toSlice(raw interface{}) ([]interface{}, bool) {
	if raw == nil {
		return nil, false
	}
	switch v := raw.(type) {
	case []interface{}:
		return v, true
	case *schema.Set:
		return v.List(), true
	default:
		return nil, false
	}
}
