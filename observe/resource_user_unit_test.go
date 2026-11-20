package observe

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

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
