package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

var entityTagsAttrType = types.SetType{ElemType: types.StringType}

// expandEntityTagsFromTFMap converts a framework types.Map (map of string sets)
// to the GraphQL EntityTagMappingInput slice.
func expandEntityTagsFromTFMap(ctx context.Context, tagsMap types.Map, diags *diag.Diagnostics) []gql.EntityTagMappingInput {
	if tagsMap.IsNull() || tagsMap.IsUnknown() || len(tagsMap.Elements()) == 0 {
		return []gql.EntityTagMappingInput{}
	}

	var raw map[string]types.Set
	diags.Append(tagsMap.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil
	}

	result := make([]gql.EntityTagMappingInput, 0, len(raw))
	for key, valSet := range raw {
		var values []string
		diags.Append(valSet.ElementsAs(ctx, &values, false)...)
		if diags.HasError() {
			return nil
		}
		result = append(result, gql.EntityTagMappingInput{
			Key:    key,
			Values: values,
		})
	}
	return result
}

// flattenEntityTagsToTFMap converts GraphQL EntityTagMapping to a framework
// types.Map where each value is a set of strings. Returns null when there are
// no tags, so that omitting entity_tags from config (null) matches the state.
func flattenEntityTagsToTFMap(tags []gql.EntityTagMapping) types.Map {
	if len(tags) == 0 {
		return types.MapNull(entityTagsAttrType)
	}
	elems := make(map[string]attr.Value, len(tags))
	for _, tag := range tags {
		vals := make([]attr.Value, len(tag.Values))
		for i, v := range tag.Values {
			vals[i] = types.StringValue(v)
		}
		set, _ := types.SetValue(types.StringType, vals)
		elems[tag.Key] = set
	}
	result, _ := types.MapValue(entityTagsAttrType, elems)
	return result
}
