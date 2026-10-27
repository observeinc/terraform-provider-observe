package observe

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func TestInboundShareErrorDiags(t *testing.T) {
	tests := []struct {
		errorType string
		hint      string
	}{
		{rest.ErrorTypeFeatureNotEnabled, "not enabled for this tenant"},
		{rest.ErrorTypeOwnershipProofFailed, "observe_inbound_share_registration_token"},
		{rest.ErrorTypeShareHasTrackedTables, "observe_inbound_share_table"},
		{rest.ErrorTypeShareMountFailed, "Apply again to retry"},
		{rest.ErrorTypeShareNotQueryable, "observe_inbound_share_setup"},
		{rest.ErrorTypeShareAlreadyMounted, "Contact Observe support"},
		{"", ""},
	}
	for _, tt := range tests {
		err := rest.ErrorWithStatusCode{StatusCode: http.StatusForbidden, Type: tt.errorType, Err: errors.New("boom")}
		diags := inboundShareErrorDiags("failed", err)
		if len(diags) != 1 || diags[0].Summary != "failed" || !strings.HasPrefix(diags[0].Detail, err.Error()) {
			t.Fatalf("%q: unexpected diagnostics %+v", tt.errorType, diags)
		}
		if tt.hint != "" && !strings.Contains(diags[0].Detail, tt.hint) {
			t.Errorf("%q: expected hint %q in %q", tt.errorType, tt.hint, diags[0].Detail)
		}
		if tt.hint == "" && diags[0].Detail != err.Error() {
			t.Errorf("expected no hint, got %q", diags[0].Detail)
		}
	}
}
