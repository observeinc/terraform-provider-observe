package observe

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

func TestFlattenEntityTagsToTFMap(t *testing.T) {
	t.Run("empty tags returns null", func(t *testing.T) {
		result := flattenEntityTagsToTFMap(nil)
		if !result.IsNull() {
			t.Fatal("expected null map for nil tags")
		}
	})

	t.Run("tags are flattened to map of sets", func(t *testing.T) {
		tags := []gql.EntityTagMapping{
			{Key: "env", Values: []string{"prod", "staging"}},
			{Key: "team", Values: []string{"backend"}},
		}
		result := flattenEntityTagsToTFMap(tags)
		if result.IsNull() {
			t.Fatal("expected non-null map")
		}
		elems := result.Elements()
		if len(elems) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(elems))
		}

		envSet, ok := elems["env"].(types.Set)
		if !ok {
			t.Fatal("expected env element to be types.Set")
		}
		if len(envSet.Elements()) != 2 {
			t.Fatalf("expected env set to have 2 elements, got %d", len(envSet.Elements()))
		}

		teamSet, ok := elems["team"].(types.Set)
		if !ok {
			t.Fatal("expected team element to be types.Set")
		}
		if len(teamSet.Elements()) != 1 {
			t.Fatalf("expected team set to have 1 element, got %d", len(teamSet.Elements()))
		}
	})
}

func TestExpandEntityTagsFromTFMap(t *testing.T) {
	ctx := context.Background()

	t.Run("null map returns empty slice", func(t *testing.T) {
		var diags diag.Diagnostics
		result := expandEntityTagsFromTFMap(ctx, types.MapNull(entityTagsAttrType), &diags)
		if diags.HasError() {
			t.Fatalf("expected no errors, got: %s", diags.Errors())
		}
		if len(result) != 0 {
			t.Fatalf("expected empty slice, got %d items", len(result))
		}
	})

	t.Run("round-trip through flatten and expand", func(t *testing.T) {
		tags := []gql.EntityTagMapping{
			{Key: "env", Values: []string{"prod"}},
			{Key: "team", Values: []string{"backend", "frontend"}},
		}
		tfMap := flattenEntityTagsToTFMap(tags)

		var diags diag.Diagnostics
		result := expandEntityTagsFromTFMap(ctx, tfMap, &diags)
		if diags.HasError() {
			t.Fatalf("expected no errors, got: %s", diags.Errors())
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tag mappings, got %d", len(result))
		}

		resultByKey := make(map[string][]string, len(result))
		for _, r := range result {
			resultByKey[r.Key] = r.Values
		}

		envVals, ok := resultByKey["env"]
		if !ok {
			t.Fatal("expected key 'env' in result")
		}
		if len(envVals) != 1 || envVals[0] != "prod" {
			t.Fatalf("expected env values [prod], got %v", envVals)
		}

		teamVals, ok := resultByKey["team"]
		if !ok {
			t.Fatal("expected key 'team' in result")
		}
		if len(teamVals) != 2 {
			t.Fatalf("expected team to have 2 values, got %d", len(teamVals))
		}
		teamSet := map[string]bool{teamVals[0]: true, teamVals[1]: true}
		if !teamSet["backend"] || !teamSet["frontend"] {
			t.Fatalf("expected team values {backend, frontend}, got %v", teamVals)
		}
	})
}
