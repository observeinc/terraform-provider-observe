package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

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
