package observe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// configureClient extracts the *observe.Client from provider data in a
// resource's Configure method. Returns nil if provider data is not yet available.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *observe.Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*observe.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *observe.Client, got: %T", req.ProviderData),
		)
		return nil
	}
	return client
}

// oidVersionPlanModifier suppresses diffs when only the OID version changes.
type oidVersionPlanModifier struct{}

func (m *oidVersionPlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs when only the OID version changes."
}

func (m *oidVersionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *oidVersionPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	oldOid, err := oid.NewOID(req.StateValue.ValueString())
	if err != nil {
		return
	}
	newOid, err := oid.NewOID(req.PlanValue.ValueString())
	if err != nil {
		return
	}
	if oldOid.Type == newOid.Type && oldOid.Id == newOid.Id {
		resp.PlanValue = req.StateValue
	}
}

// oidTypeValidator validates that a string value is a valid OID of one of the given types.
type oidTypeValidator struct {
	types []oid.Type
}

func validateFWOID(types ...oid.Type) validator.String {
	return &oidTypeValidator{types: types}
}

func (v *oidTypeValidator) Description(_ context.Context) string {
	names := make([]string, len(v.types))
	for i, t := range v.types {
		names[i] = string(t)
	}
	return fmt.Sprintf("value must be an OID of type: %s", strings.Join(names, ", "))
}

func (v *oidTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *oidTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	id, err := oid.NewOID(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid OID", err.Error())
		return
	}

	for _, t := range v.types {
		if id.Type == t {
			return
		}
	}

	names := make([]string, len(v.types))
	for i, t := range v.types {
		names[i] = string(t)
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Wrong OID type",
		fmt.Sprintf("OID type must be %s, got %s", strings.Join(names, ", "), id.Type),
	)
}

type timeDurationValidator struct{}

func validateFWTimeDuration() validator.String {
	return &timeDurationValidator{}
}

func (v *timeDurationValidator) Description(_ context.Context) string {
	return "value must be a valid Go duration string (e.g. 3s, 2m, 1h)"
}

func (v *timeDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *timeDurationValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.ParseDuration(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid duration", err.Error())
	}
}

type flagsValidator struct{}

func validateFWFlags() validator.String {
	return &flagsValidator{}
}

func (v *flagsValidator) Description(_ context.Context) string {
	return "value must be a valid flags string"
}

func (v *flagsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *flagsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := convertFlags(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid flags", err.Error())
	}
}

// enumValidator validates that a string is one of a set of allowed values (case-insensitive via snake_case normalization).
type enumValidator struct {
	allowed []string
}

func validateFWEnums(stringerSlice interface{}) validator.String {
	return &enumValidator{allowed: snakeCased(stringerSlice)}
}

func (v *enumValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(v.allowed, ", "))
}

func (v *enumValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *enumValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := toSnake(req.ConfigValue.ValueString())
	for _, a := range v.allowed {
		if strings.EqualFold(val, a) {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		fmt.Sprintf("must be one of: %s, got: %s", strings.Join(v.allowed, ", "), req.ConfigValue.ValueString()),
	)
}
