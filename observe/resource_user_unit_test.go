package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

func TestResourceUserSchemaValid(t *testing.T) {
	if err := resourceUser().InternalValidate(nil, true); err != nil {
		t.Fatalf("schema: %s", err)
	}
}

func TestNewUserCreateInputMinimal(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceUser().Schema, map[string]interface{}{
		"email":             "jane@example.com",
		"label":             "Jane Doe",
		"identity_provider": "SAML2",
		"identity":          "jane@example.com",
	})

	input, diags := newUserCreateInput(data)
	if diags.HasError() {
		t.Fatalf("build input: %v", diags)
	}

	// Unset optional fields must be omitted entirely, not sent as null.
	b, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"email":"jane@example.com","label":"Jane Doe","ssoIdentities":[{"identityProvider":"SAML2","identity":"jane@example.com"}]}`
	if got := string(b); got != want {
		t.Errorf("input JSON =\n  %s\nwant\n  %s", got, want)
	}
}

func TestNewUserCreateInputAllFields(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceUser().Schema, map[string]interface{}{
		"email":             "jane@example.com",
		"label":             "Jane Doe",
		"identity_provider": "GOOGLE",
		"identity":          "1234567890",
		"role":              "writer",
		"comment":           "contractor",
		"expiration_time":   "2027-01-01T00:00:00Z",
	})

	input, diags := newUserCreateInput(data)
	if diags.HasError() {
		t.Fatalf("build input: %v", diags)
	}
	if input.Role == nil || *input.Role != "writer" {
		t.Errorf("role = %v, want writer", input.Role)
	}
	if input.Comment == nil || *input.Comment != "contractor" {
		t.Errorf("comment = %v, want contractor", input.Comment)
	}
	if input.ExpirationTime == nil || input.ExpirationTime.String() != "2027-01-01T00:00:00Z" {
		t.Errorf("expirationTime = %v", input.ExpirationTime)
	}
	if got := input.SsoIdentities[0].IdentityProvider; got != gql.SSOIdentityProviderGoogle {
		t.Errorf("identityProvider = %q, want GOOGLE", got)
	}
}

func TestDisableUserInputSendsOnlyStatus(t *testing.T) {
	status := gql.UserStatusUserstatusdisabled
	b, err := json.Marshal(&gql.UserInput{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"status":"UserStatusDisabled"}`; got != want {
		t.Errorf("input JSON = %s, want %s", got, want)
	}
}

func TestUserResourceToResourceData(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceUser().Schema, map[string]interface{}{})
	data.SetId("123")

	comment := "contractor"
	exp := types.TimeScalar(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	diags := userResourceToResourceData(&gql.User{
		Id:             types.UserIdScalar(123),
		Email:          "jane@example.com",
		Label:          "Jane Doe",
		Role:           "reader",
		Status:         gql.UserStatusUserstatusdisabled,
		Comment:        &comment,
		ExpirationTime: &exp,
		SsoIdentities: []gql.UserSsoIdentitiesSSOIdentity{{
			IdentityProvider: gql.SSOIdentityProviderSaml2,
			Identity:         "jane@example.com",
		}},
	}, data)
	if diags.HasError() {
		t.Fatalf("map user: %v", diags)
	}

	for k, want := range map[string]interface{}{
		"email":             "jane@example.com",
		"label":             "Jane Doe",
		"role":              "reader",
		"status":            "UserStatusDisabled",
		"disabled":          true,
		"comment":           "contractor",
		"expiration_time":   "2027-01-01T00:00:00Z",
		"identity_provider": "SAML2",
		"identity":          "jane@example.com",
		"oid":               "o:::user:123",
	} {
		if got := data.Get(k); got != want {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}
}

// userDiff plans config against an existing user's state, running CustomizeDiff.
func userDiff(t *testing.T, state, config map[string]interface{}) (*terraform.InstanceDiff, error) {
	t.Helper()
	r := resourceUser()
	attrs := map[string]string{}
	for k, v := range state {
		attrs[k] = fmt.Sprint(v)
	}
	is := &terraform.InstanceState{ID: "123", Attributes: attrs}
	return r.Diff(context.Background(), is, terraform.NewResourceConfigRaw(config), nil)
}

var userBaseConfig = map[string]interface{}{
	"email":             "jane@example.com",
	"label":             "Jane Doe",
	"identity_provider": "SAML2",
	"identity":          "jane@example.com",
}

func userConfig(overrides map[string]interface{}) map[string]interface{} {
	c := map[string]interface{}{}
	for k, v := range userBaseConfig {
		c[k] = v
	}
	for k, v := range overrides {
		c[k] = v
	}
	return c
}

func TestUserIdentityProviderChangeIsInPlace(t *testing.T) {
	state := userConfig(map[string]interface{}{"disabled": false})
	d, err := userDiff(t, state, userConfig(map[string]interface{}{"identity_provider": "GOOGLE"}))
	if err != nil {
		t.Fatal(err)
	}
	if d.RequiresNew() {
		t.Error("changing identity_provider must update in place, not replace")
	}
}

func TestUserExpirationRemovalRejectedAtPlan(t *testing.T) {
	state := userConfig(map[string]interface{}{"disabled": false, "expiration_time": "2027-01-01T00:00:00Z"})
	if _, err := userDiff(t, state, userConfig(nil)); err == nil || !strings.Contains(err.Error(), "expiration_time") {
		t.Errorf("err = %v, want expiration_time removal error", err)
	}
	// Changing it to a new value is fine.
	if _, err := userDiff(t, state, userConfig(map[string]interface{}{"expiration_time": "2028-01-01T00:00:00Z"})); err != nil {
		t.Errorf("updating expiration_time: %v", err)
	}
}

func TestNewUserUpdateInputSendsIdentityOnProviderChange(t *testing.T) {
	r := resourceUser()
	state := &terraform.InstanceState{ID: "123", Attributes: map[string]string{
		"email": "jane@example.com", "label": "Jane Doe", "identity_provider": "SAML2",
		"identity": "jane@example.com", "disabled": "false",
	}}
	d, err := userDiff(t, userConfig(map[string]interface{}{"disabled": false}), userConfig(map[string]interface{}{"identity_provider": "GOOGLE"}))
	if err != nil {
		t.Fatal(err)
	}
	data, err := schema.InternalMap(r.Schema).Data(state, d)
	if err != nil {
		t.Fatal(err)
	}
	input, changed, diags := newUserUpdateInput(data)
	if diags.HasError() || !changed {
		t.Fatalf("changed=%v diags=%v", changed, diags)
	}
	if len(input.SsoIdentities) != 1 || input.SsoIdentities[0].IdentityProvider != gql.SSOIdentityProviderGoogle {
		t.Errorf("ssoIdentities = %+v, want GOOGLE", input.SsoIdentities)
	}
}

func TestUserExpirationUnknownIsNotRemoval(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	state := userConfig(map[string]interface{}{"disabled": false, "expiration_time": "2027-01-01T00:00:00Z"})
	cfg := userConfig(map[string]interface{}{"expiration_time": unknown})
	if _, err := userDiff(t, state, cfg); err != nil {
		t.Errorf("unknown expiration_time must not be treated as removal: %v", err)
	}
}
