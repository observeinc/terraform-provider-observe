package observe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

// inboundShareGetResponse matches the GetShare fixture in client/rest.
const inboundShareGetResponse = `{
	"id": "41012345",
	"shareName": "ACME_CUSTOMER_DATA",
	"providerType": "Snowflake",
	"snowflakeConfig": {
		"shareName": "CUSTOMER_SHARE_PROD",
		"providerAccount": "ACME_CORP.US-EAST-1"
	},
	"status": {
		"state": "Active",
		"health": "Healthy"
	},
	"createdBy": {"id": "123"},
	"createdAt": "2026-01-15T10:30:00Z",
	"updatedBy": {"id": "123"},
	"updatedAt": "2026-03-20T14:22:00Z",
	"tableCount": 15
}`

func newInboundShareTestClient(server *httptest.Server) *observeclient.Client {
	return &observeclient.Client{Config: &observeclient.Config{}, Rest: rest.New(server.URL, server.Client())}
}

func TestResourceInboundShareImportReadsSnowflakeConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/shares/inbound/41012345" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(inboundShareGetResponse))
	}))
	defer server.Close()
	client := newInboundShareTestClient(server)

	data := resourceInboundShare().TestResourceData()
	data.SetId("o:::inboundshare:41012345")
	imported, err := resourceInboundShareImport(context.Background(), data, client)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if diags := resourceInboundShareRead(context.Background(), imported[0], client); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if got := imported[0].Get("share_name"); got != "CUSTOMER_SHARE_PROD" {
		t.Errorf("share_name = %q, want CUSTOMER_SHARE_PROD", got)
	}
	if got := imported[0].Get("provider_account"); got != "ACME_CORP.US-EAST-1" {
		t.Errorf("provider_account = %q, want ACME_CORP.US-EAST-1", got)
	}
}

func TestResourceInboundShareCreateRejectsExistingShare(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(inboundShareGetResponse))
	}))
	defer server.Close()
	client := newInboundShareTestClient(server)

	data := schema.TestResourceDataRaw(t, resourceInboundShare().Schema, map[string]interface{}{
		"share_name":       "CUSTOMER_SHARE_PROD",
		"provider_account": "ACME_CORP.US-EAST-1",
		"label":            "Customer data",
	})
	diags := resourceInboundShareCreate(context.Background(), data, client)
	if !diags.HasError() || !strings.Contains(diags[0].Detail, "terraform import") || !strings.Contains(diags[0].Detail, "41012345") {
		t.Fatalf("expected an import-it-instead error, got %v", diags)
	}
	if data.Id() != "" {
		t.Errorf("state ID = %q, want none", data.Id())
	}
	if len(methods) != 1 || methods[0] != http.MethodPost {
		t.Errorf("requests = %v, want only the register POST", methods)
	}
}

func TestResourceInboundShareUnsetLabelResetsToShareName(t *testing.T) {
	var patched map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&patched); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "41012345", "shareName": "CUSTOMER_SHARE_PROD", "label": "CUSTOMER_SHARE_PROD", "providerType": "Snowflake"}`))
	}))
	defer server.Close()
	client := newInboundShareTestClient(server)

	data := schema.TestResourceDataRaw(t, resourceInboundShare().Schema, map[string]interface{}{
		"share_name":       "CUSTOMER_SHARE_PROD",
		"provider_account": "ACME_CORP.US-EAST-1",
	})
	data.SetId("41012345")
	if diags := resourceInboundShareUpdate(context.Background(), data, client); diags.HasError() {
		t.Fatalf("update: %v", diags)
	}
	if patched["label"] != "CUSTOMER_SHARE_PROD" {
		t.Errorf("PATCH body = %v, want label CUSTOMER_SHARE_PROD", patched)
	}
	if got := data.Get("label"); got != "" {
		t.Errorf("label = %q, want it left unset while the server reports the default", got)
	}
}

func TestInboundShareToResourceDataKeepsCustomLabel(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceInboundShare().Schema, map[string]interface{}{
		"share_name":       "CUSTOMER_SHARE_PROD",
		"provider_account": "ACME_CORP.US-EAST-1",
	})
	share := &rest.Share{Id: "41012345", ShareName: "CUSTOMER_SHARE_PROD", Label: "Renamed in the UI"}
	if diags := inboundShareToResourceData(share, data); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	if got := data.Get("label"); got != "Renamed in the UI" {
		t.Errorf("label = %q, want the server's custom label so the plan resets it", got)
	}
}
