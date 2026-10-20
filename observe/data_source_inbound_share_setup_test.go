package observe

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func TestAccObserveInboundShareSetup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `data "observe_inbound_share_setup" "setup" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "consumer_account"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "region"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "summary"),
					resource.TestCheckResourceAttrSet("data.observe_inbound_share_setup.setup", "setup_sql.0.sql"),
				),
			},
		},
	})
}

func TestInboundShareErrorDiags(t *testing.T) {
	tests := []struct {
		errorType string
		hint      string
	}{
		{rest.ErrorTypeFeatureNotEnabled, "not enabled for this tenant"},
		{rest.ErrorTypeOwnershipProofFailed, "observe_inbound_share_registration_token"},
		{rest.ErrorTypeShareHasTrackedTables, "observe_inbound_share_table"},
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

func TestResourceInboundShareImportAcceptsOID(t *testing.T) {
	for _, id := range []string{"41012345", "o:::inboundshare:41012345"} {
		data := resourceInboundShare().TestResourceData()
		data.SetId(id)
		result, err := resourceInboundShareImport(context.Background(), data, nil)
		if err != nil {
			t.Fatalf("import %q failed: %v", id, err)
		}
		if got := result[0].Id(); got != "41012345" {
			t.Errorf("import %q: expected ID 41012345, got %s", id, got)
		}
	}
}
