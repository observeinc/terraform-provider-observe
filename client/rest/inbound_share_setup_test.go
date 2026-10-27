package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSetupInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/setup" || r.Method != http.MethodGet {
			t.Errorf("Expected GET /v1/shares/inbound/setup, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"consumerAccount": "OBSERVE.CONSUMER",
			"region": "AWS_US_WEST_2",
			"cloud": "AWS",
			"guidance": {
				"summary": "Share a database with Observe.",
				"setupSql": [{"description": "Create the share", "sql": "CREATE SHARE <your_share_name>;"}],
				"providerChecks": [{"description": "Check grants", "sql": "SHOW GRANTS TO SHARE <your_share_name>;"}],
				"checklist": ["Change tracking is on"]
			}
		}`))
	}))
	defer server.Close()

	info, err := New(server.URL, server.Client()).GetSetupInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetupInfo failed: %v", err)
	}
	if info.ConsumerAccount != "OBSERVE.CONSUMER" {
		t.Errorf("Expected consumer account OBSERVE.CONSUMER, got %s", info.ConsumerAccount)
	}
	if len(info.Guidance.SetupSql) != 1 || info.Guidance.SetupSql[0].Sql != "CREATE SHARE <your_share_name>;" {
		t.Errorf("Unexpected setup SQL: %+v", info.Guidance.SetupSql)
	}
	if len(info.Guidance.Checklist) != 1 {
		t.Errorf("Expected one checklist item, got %d", len(info.Guidance.Checklist))
	}
}

func TestGetSetupInfoMissingGuidance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"consumerAccount": "OBSERVE.CONSUMER", "region": "AWS_US_WEST_2", "cloud": null}`))
	}))
	defer server.Close()

	_, err := New(server.URL, server.Client()).GetSetupInfo(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing guidance") {
		t.Fatalf("Expected a missing guidance error, got %v", err)
	}
}

func TestCreateRegistrationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/registration-tokens" || r.Method != http.MethodPost {
			t.Errorf("Expected POST /v1/shares/inbound/registration-tokens, got %s %s", r.Method, r.URL.Path)
		}
		var req CreateRegistrationTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProviderAccount != "ACME.PROD" {
			t.Errorf("Unexpected request body: %+v (%v)", req, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"token": "obs-share-abc",
			"expiresAt": "2026-10-21T00:00:00Z",
			"setupSql": [{"description": "Set the comment", "sql": "ALTER SHARE <your_share_name> SET COMMENT = 'obs-share-abc';"}]
		}`))
	}))
	defer server.Close()

	token, err := New(server.URL, server.Client()).CreateRegistrationToken(context.Background(), &CreateRegistrationTokenRequest{ProviderAccount: "ACME.PROD"})
	if err != nil {
		t.Fatalf("CreateRegistrationToken failed: %v", err)
	}
	if token.Token != "obs-share-abc" || token.ExpiresAt != "2026-10-21T00:00:00Z" {
		t.Errorf("Unexpected token: expiresAt=%s", token.ExpiresAt)
	}
	if len(token.SetupSql) != 1 || !strings.Contains(token.SetupSql[0].Sql, "SET COMMENT") {
		t.Errorf("Unexpected setup SQL: %+v", token.SetupSql)
	}
}

func TestRegisterShareOwnershipProofFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound" || r.Method != http.MethodPost {
			t.Errorf("Expected POST /v1/shares/inbound, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"type": "ownership_proof_failed", "message": "share comment does not hold a valid token"}`))
	}))
	defer server.Close()

	_, _, err := New(server.URL, server.Client()).RegisterShare(context.Background(), &RegisterShareRequest{ProviderAccount: "ACME.PROD", ShareName: "SALES"})
	if !HasStatusCode(err, http.StatusForbidden) || !HasErrorType(err, ErrorTypeOwnershipProofFailed) {
		t.Fatalf("Expected 403 ownership_proof_failed, got %v", err)
	}
}

func TestRegisterShareReportsCreated(t *testing.T) {
	for _, tt := range []struct {
		status  int
		created bool
	}{
		{http.StatusCreated, true},
		{http.StatusOK, false},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tt.status)
			_, _ = w.Write([]byte(`{"id": "41012345", "shareName": "SALES", "providerType": "Snowflake"}`))
		}))

		share, created, err := New(server.URL, server.Client()).RegisterShare(context.Background(), &RegisterShareRequest{ProviderAccount: "ACME.PROD", ShareName: "SALES"})
		server.Close()
		if err != nil {
			t.Fatalf("status %d: RegisterShare failed: %v", tt.status, err)
		}
		if share.Id != "41012345" || created != tt.created {
			t.Errorf("status %d: got id %s created %v, want created %v", tt.status, share.Id, created, tt.created)
		}
	}
}

func TestDeleteShareForce(t *testing.T) {
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345" || r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE /v1/shares/inbound/41012345, got %s %s", r.Method, r.URL.Path)
		}
		queries = append(queries, r.URL.RawQuery)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	if err := client.DeleteShare(context.Background(), "41012345", false); err != nil {
		t.Fatalf("DeleteShare failed: %v", err)
	}
	if err := client.DeleteShare(context.Background(), "41012345", true); err != nil {
		t.Fatalf("DeleteShare with force failed: %v", err)
	}
	if len(queries) != 2 || queries[0] != "" || queries[1] != "force=true" {
		t.Errorf("Unexpected queries: %v", queries)
	}
}
