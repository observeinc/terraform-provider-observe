package observe

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func TestOIDVersionPlanModifier(t *testing.T) {
	ctx := context.Background()
	mod := &oidVersionPlanModifier{}

	t.Run("same id different version is suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("o:::dataset:123/2020-01-16T21:06:19Z"),
			PlanValue:  types.StringValue("o:::dataset:123/2021-03-10T10:00:00Z"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatalf("expected plan to be suppressed to state value %s, got %s", req.StateValue, resp.PlanValue)
		}
	})

	t.Run("different id is not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("o:::dataset:123/2020-01-16T21:06:19Z"),
			PlanValue:  types.StringValue("o:::dataset:456/2020-01-16T21:06:19Z"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatalf("expected plan value to remain %s, got %s", req.PlanValue, resp.PlanValue)
		}
	})

	t.Run("different type is not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("o:::dataset:123/2020-01-16T21:06:19Z"),
			PlanValue:  types.StringValue("o:::dashboard:123/2020-01-16T21:06:19Z"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatalf("expected plan value to remain %s, got %s", req.PlanValue, resp.PlanValue)
		}
	})

	t.Run("null state is not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringNull(),
			PlanValue:  types.StringValue("o:::dataset:123/2020-01-16T21:06:19Z"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatalf("expected plan value to remain %s, got %s", req.PlanValue, resp.PlanValue)
		}
	})
}

func TestValidateFWOID(t *testing.T) {
	ctx := context.Background()

	t.Run("valid bookmark group OID", func(t *testing.T) {
		v := validateFWOID(oid.TypeBookmarkGroup)
		req := validator.StringRequest{
			ConfigValue: types.StringValue("o:::bookmarkgroup:123"),
			Path:        path.Root("group"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("wrong OID type", func(t *testing.T) {
		v := validateFWOID(oid.TypeBookmarkGroup)
		req := validator.StringRequest{
			ConfigValue: types.StringValue("o:::dataset:123"),
			Path:        path.Root("group"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for wrong OID type")
		}
	})

	t.Run("multiple allowed types", func(t *testing.T) {
		v := validateFWOID(oid.TypeDataset, oid.TypeDashboard)
		for _, valid := range []string{"o:::dataset:123", "o:::dashboard:456"} {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(valid),
				Path:        path.Root("target"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected %s to be valid, got: %s", valid, resp.Diagnostics.Errors())
			}
		}
	})

	t.Run("invalid OID string", func(t *testing.T) {
		v := validateFWOID(oid.TypeDataset)
		req := validator.StringRequest{
			ConfigValue: types.StringValue("not-an-oid"),
			Path:        path.Root("target"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for invalid OID string")
		}
	})

	t.Run("null is skipped", func(t *testing.T) {
		v := validateFWOID(oid.TypeDataset)
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("target"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for null value, got: %s", resp.Diagnostics.Errors())
		}
	})
}

func TestValidateFWEnums(t *testing.T) {
	ctx := context.Background()
	v := validateFWEnums(gql.AllBookmarkKindTypes)

	t.Run("valid snake_case value", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("log_explorer"),
			Path:        path.Root("bookmark_kind"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("nonexistent"),
			Path:        path.Root("bookmark_kind"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for invalid enum value")
		}
	})

	t.Run("null is skipped", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("bookmark_kind"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for null value, got: %s", resp.Diagnostics.Errors())
		}
	})
}

func TestValidateFWTimeDuration(t *testing.T) {
	ctx := context.Background()
	v := validateFWTimeDuration()

	t.Run("valid duration", func(t *testing.T) {
		for _, d := range []string{"3s", "2m", "1h30m", "500ms"} {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(d),
				Path:        path.Root("retry_wait"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected %s to be valid, got: %s", d, resp.Diagnostics.Errors())
			}
		}
	})

	t.Run("invalid duration", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("not-a-duration"),
			Path:        path.Root("retry_wait"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for invalid duration")
		}
	})

	t.Run("null is skipped", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("retry_wait"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for null, got: %s", resp.Diagnostics.Errors())
		}
	})
}

func TestValidateFWFlags(t *testing.T) {
	ctx := context.Background()
	v := validateFWFlags()

	t.Run("valid flags", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("cache-client"),
			Path:        path.Root("flags"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("empty string is valid", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue(""),
			Path:        path.Root("flags"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("null is skipped", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("flags"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for null, got: %s", resp.Diagnostics.Errors())
		}
	})
}

func TestValidateFWJSON(t *testing.T) {
	ctx := context.Background()
	v := validateFWJSON()

	t.Run("valid JSON", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue(`{"key": "value"}`),
			Path:        path.Root("data"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue(`{broken`),
			Path:        path.Root("data"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for invalid JSON")
		}
	})
}

func TestValidateFWDatasetName(t *testing.T) {
	ctx := context.Background()
	v := validateFWDatasetName()

	t.Run("valid name", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("my-dataset"),
			Path:        path.Root("name"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("empty name", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue(""),
			Path:        path.Root("name"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for empty name")
		}
	})

	t.Run("invalid characters", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("bad:name"),
			Path:        path.Root("name"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for name with colon")
		}
	})
}

func TestValidateFWStringNotEmpty(t *testing.T) {
	ctx := context.Background()
	v := validateFWStringNotEmpty()

	t.Run("non-empty is valid", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("hello"),
			Path:        path.Root("val"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("empty string is invalid", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue(""),
			Path:        path.Root("val"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for empty string")
		}
	})
}

func TestValidateFWRegex(t *testing.T) {
	ctx := context.Background()
	v := validateFWRegex(`^\d+$`, "expected valid integer")

	t.Run("matching string", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("12345"),
			Path:        path.Root("id"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors, got: %s", resp.Diagnostics.Errors())
		}
	})

	t.Run("non-matching string", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringValue("abc"),
			Path:        path.Root("id"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for non-matching string")
		}
	})
}

func TestPipelinePlanModifier(t *testing.T) {
	ctx := context.Background()
	mod := &pipelinePlanModifier{}

	t.Run("trailing whitespace is suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("make_col x:1\n"),
			PlanValue:  types.StringValue("make_col x:1\n\t\n"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatalf("expected suppression, got %q", resp.PlanValue.ValueString())
		}
	})

	t.Run("different content is not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("make_col x:1"),
			PlanValue:  types.StringValue("make_col x:2"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatal("expected no suppression for different content")
		}
	})
}

func TestTimeDurationPlanModifier(t *testing.T) {
	ctx := context.Background()

	t.Run("equivalent durations are suppressed", func(t *testing.T) {
		mod := &timeDurationPlanModifier{}
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("1m0s"),
			PlanValue:  types.StringValue("60s"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatal("expected suppression for equivalent durations")
		}
	})

	t.Run("ceil days mode", func(t *testing.T) {
		mod := &timeDurationPlanModifier{ceilDays: true}
		// 48h39s and 72h0m0s both ceil to 72h (3 days)
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("72h0m0s"),
			PlanValue:  types.StringValue("48h39s"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatal("expected suppression for durations that ceil to same day count")
		}
	})

	t.Run("different durations are not suppressed", func(t *testing.T) {
		mod := &timeDurationPlanModifier{}
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("1m"),
			PlanValue:  types.StringValue("2m"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatal("expected no suppression for different durations")
		}
	})
}

func TestJSONPlanModifier(t *testing.T) {
	ctx := context.Background()
	mod := &jsonPlanModifier{}

	t.Run("semantically equal JSON is suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue(`{"a":"b","c":"d"}`),
			PlanValue:  types.StringValue(`{"c":"d","a":"b"}`),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatal("expected suppression for semantically equal JSON")
		}
	})

	t.Run("different JSON is not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue(`{"a":"b"}`),
			PlanValue:  types.StringValue(`{"a":"c"}`),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatal("expected no suppression for different JSON")
		}
	})
}

func TestEnumPlanModifier(t *testing.T) {
	ctx := context.Background()
	mod := &enumPlanModifier{}

	t.Run("same enum different casing is suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("skip_rematerialization"),
			PlanValue:  types.StringValue("SkipRematerialization"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.StateValue {
			t.Fatal("expected suppression for same enum with different casing")
		}
	})

	t.Run("different enum values are not suppressed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			StateValue: types.StringValue("rematerialize"),
			PlanValue:  types.StringValue("skip_rematerialization"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		mod.PlanModifyString(ctx, req, resp)
		if resp.PlanValue != req.PlanValue {
			t.Fatal("expected no suppression for different enum values")
		}
	})
}
