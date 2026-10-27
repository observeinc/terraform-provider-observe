package observe

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// canonicalInboundShareName is the form the API stores a share name in:
// unquoted names are upper-cased, quoted names lose their outer quotes.
func canonicalInboundShareName(name string) string {
	s := strings.TrimSpace(name)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1]
	}
	return strings.ToUpper(s)
}

// canonicalInboundShareProviderAccount is the form the API stores a provider
// account in.
func canonicalInboundShareProviderAccount(account string) string {
	return strings.ToUpper(strings.TrimSpace(account))
}

// inboundShareStateName is the spelling of a server-reported share name kept
// in state. It must canonicalize back to the same name, so a case-sensitive
// name is quoted and is never mistaken for an unquoted one.
func inboundShareStateName(name string) string {
	if name == "" || strings.ToUpper(name) == name {
		return name
	}
	return `"` + name + `"`
}

// suppressCanonicalDiff ignores a change between two spellings of the same
// name: state holds a spelling that canonicalizes to the server's name, so
// only a change of canonical form is a change.
func suppressCanonicalDiff(canonical func(string) string) schema.SchemaDiffSuppressFunc {
	return func(_, old, new string, _ *schema.ResourceData) bool {
		if old == "" || new == "" {
			return false
		}
		return canonical(old) == canonical(new)
	}
}

// inboundShareStatusSchema holds the computed attributes that the
// observe_inbound_share resource and data source share.
func inboundShareStatusSchema() map[string]*schema.Schema {
	computed := func(t schema.ValueType, key string) *schema.Schema {
		return &schema.Schema{
			Type:        t,
			Computed:    true,
			Description: descriptions.Get("inbound_share", "schema", key),
		}
	}
	return map[string]*schema.Schema{
		"oid": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: descriptions.Get("common", "schema", "oid"),
		},
		"provider_type":     computed(schema.TypeString, "provider_type"),
		"status":            computed(schema.TypeString, "status"),
		"health":            computed(schema.TypeString, "health"),
		"health_message":    computed(schema.TypeString, "health_message"),
		"last_health_check": computed(schema.TypeString, "last_health_check"),
		"table_count":       computed(schema.TypeInt, "table_count"),
		"created_at":        computed(schema.TypeString, "created_at"),
		"updated_at":        computed(schema.TypeString, "updated_at"),
	}
}

// inboundShareStatusValues is the state for inboundShareStatusSchema.
func inboundShareStatusValues(s *rest.Share) map[string]interface{} {
	healthMessage := ""
	if s.Status.HealthMessage != nil {
		healthMessage = *s.Status.HealthMessage
	}
	lastHealthCheck := ""
	if s.Status.LastHealthCheckAt != nil {
		lastHealthCheck = *s.Status.LastHealthCheckAt
	}
	return map[string]interface{}{
		"oid":               s.Oid().String(),
		"provider_type":     s.ProviderType,
		"status":            s.Status.State,
		"health":            s.Status.Health,
		"health_message":    healthMessage,
		"last_health_check": lastHealthCheck,
		"table_count":       s.TableCount,
		"created_at":        s.CreatedAt,
		"updated_at":        s.UpdatedAt,
	}
}

func setResourceDataValues(data *schema.ResourceData, values map[string]interface{}) (diags diag.Diagnostics) {
	for key, value := range values {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	return diags
}

// inboundShareAlreadyRegisteredDiags rejects a create for a share that Observe
// already has, so that destroying this resource cannot drop a share it did not
// create.
func inboundShareAlreadyRegisteredDiags(s *rest.Share, shareName, providerAccount string) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "inbound share is already registered",
		Detail: fmt.Sprintf("Share %q from provider account %q is already registered in Observe with ID %s. "+
			"Import it instead: terraform import <resource address> %s", shareName, providerAccount, s.Id, s.Id),
	}}
}

// inboundShareErrorDiags wraps an inbound share API error, adding what to do
// next for the error types that have a known fix.
func inboundShareErrorDiags(summary string, err error) diag.Diagnostics {
	d := diag.Diagnostic{
		Severity: diag.Error,
		Summary:  summary,
		Detail:   err.Error(),
	}
	switch {
	case rest.HasErrorType(err, rest.ErrorTypeFeatureNotEnabled):
		d.Detail += "\n\nInbound shares are not enabled for this tenant. Contact Observe support to enable them."
	case rest.HasErrorType(err, rest.ErrorTypeOwnershipProofFailed):
		d.Detail += "\n\nCheck that the share is granted to Observe's consumer account (see the observe_inbound_share_setup data source) and that its comment is set to an unexpired token from observe_inbound_share_registration_token."
	case rest.HasErrorType(err, rest.ErrorTypeShareHasTrackedTables):
		d.Detail += "\n\nDestroy the observe_inbound_share_table resources for this share first."
	}
	return diag.Diagnostics{d}
}
