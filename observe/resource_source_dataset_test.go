package observe

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// observe_source_dataset is a deprecated no-op. These tests pass a nil meta, so
// any attempt to reach the Observe client would panic.
func TestSourceDatasetResourceIsNoOp(t *testing.T) {
	ctx := context.Background()
	r := resourceSourceDataset()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("invalid resource schema: %s", err)
	}
	if r.DeprecationMessage == "" {
		t.Error("expected a deprecation message")
	}

	existing := func() *schema.ResourceData {
		d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
			"name":                     "existing",
			"schema":                   "EXTERNAL",
			"table_name":               "T",
			"source_update_table_name": "T_UPDATES",
			"valid_from_field":         "TIMESTAMP",
			"field": []interface{}{map[string]interface{}{
				"name":     "TIMESTAMP",
				"type":     "timestamp",
				"sql_type": "NUMBER(38,0)",
			}},
		})
		d.SetId("41000001")
		if err := d.Set("oid", "o:::dataset:41000001/1"); err != nil {
			t.Fatal(err)
		}
		return d
	}

	onlyWarnings := func(t *testing.T, diags diag.Diagnostics) {
		t.Helper()
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags)
		}
		if len(diags) == 0 {
			t.Fatal("expected a warning")
		}
	}

	t.Run("create fails", func(t *testing.T) {
		d := existing()
		d.SetId("")
		if diags := r.CreateContext(ctx, d, nil); !diags.HasError() {
			t.Fatalf("expected create to fail, got %v", diags)
		}
	})

	t.Run("read keeps state", func(t *testing.T) {
		d := existing()
		if diags := r.ReadContext(ctx, d, nil); len(diags) != 0 {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if d.Id() != "41000001" || d.Get("oid") != "o:::dataset:41000001/1" {
			t.Errorf("state changed: id=%q oid=%q", d.Id(), d.Get("oid"))
		}
	})

	t.Run("update warns", func(t *testing.T) {
		onlyWarnings(t, r.UpdateContext(ctx, existing(), nil))
	})

	t.Run("delete warns", func(t *testing.T) {
		onlyWarnings(t, r.DeleteContext(ctx, existing(), nil))
	})
}
