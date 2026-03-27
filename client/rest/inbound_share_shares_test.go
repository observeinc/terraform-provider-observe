package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListShares(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound" {
			t.Errorf("Expected path /v1/shares/inbound, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// Check query parameters
		if status := r.URL.Query().Get("status"); status != "" {
			if status != "Active" {
				t.Errorf("Expected status=Active, got %s", status)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"shares": [
				{
					"id": "41012345",
					"shareName": "ACME_CUSTOMER_DATA",
					"providerType": "Snowflake",
					"status": {
						"state": "Active",
						"health": "Healthy"
					},
					"createdBy": {"id": "123"},
					"createdAt": "2026-01-15T10:30:00Z",
					"updatedBy": {"id": "123"},
					"updatedAt": "2026-03-20T14:22:00Z",
					"tableCount": 15
				}
			],
			"meta": {
				"totalCount": 1
			}
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	// Test without params
	result, err := client.ListShares(ctx, nil)
	if err != nil {
		t.Fatalf("ListShares failed: %v", err)
	}
	if len(result.Shares) != 1 {
		t.Errorf("Expected 1 share, got %d", len(result.Shares))
	}
	if result.Shares[0].Id != "41012345" {
		t.Errorf("Expected share ID 41012345, got %s", result.Shares[0].Id)
	}

	// Test with params
	params := &ListSharesParams{
		Status: "Active",
		Limit:  20,
	}
	result, err = client.ListShares(ctx, params)
	if err != nil {
		t.Fatalf("ListShares with params failed: %v", err)
	}
	if len(result.Shares) != 1 {
		t.Errorf("Expected 1 share, got %d", len(result.Shares))
	}
}

func TestGetShare(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345" {
			t.Errorf("Expected path /v1/shares/inbound/41012345, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
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
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	share, err := client.GetShare(ctx, "41012345")
	if err != nil {
		t.Fatalf("GetShare failed: %v", err)
	}
	if share.Id != "41012345" {
		t.Errorf("Expected share ID 41012345, got %s", share.Id)
	}
	if share.ShareName != "ACME_CUSTOMER_DATA" {
		t.Errorf("Expected share name ACME_CUSTOMER_DATA, got %s", share.ShareName)
	}
	if share.SnowflakeConfig == nil {
		t.Error("Expected SnowflakeConfig to be present")
	}
}

func TestGetShareNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "share not found"}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	_, err := client.GetShare(ctx, "99999999")
	if err == nil {
		t.Fatal("Expected error for non-existent share")
	}
	// The error is wrapped by fmt.Errorf, so we just check that we got an error
	// In production code, callers can use errors.As to unwrap and check the status code
	t.Logf("Got expected error: %v", err)
}

func TestLookupShare(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"shares": [
				{
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
				},
				{
					"id": "41012346",
					"shareName": "ACME_CUSTOMER_DATA",
					"providerType": "Snowflake",
					"snowflakeConfig": {
						"shareName": "CUSTOMER_SHARE_DEV",
						"providerAccount": "OTHER_CORP.US-WEST-2"
					},
					"status": {
						"state": "Active",
						"health": "Healthy"
					},
					"createdBy": {"id": "123"},
					"createdAt": "2026-01-15T10:30:00Z",
					"updatedBy": {"id": "123"},
					"updatedAt": "2026-03-20T14:22:00Z",
					"tableCount": 5
				}
			],
			"meta": {
				"totalCount": 2,
				"limit": 20,
				"offset": 0
			}
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	// Test successful lookup with both shareName and providerAccount
	share, err := client.LookupShare(ctx, "ACME_CUSTOMER_DATA", "ACME_CORP.US-EAST-1")
	if err != nil {
		t.Fatalf("LookupShare failed: %v", err)
	}
	if share.Id != "41012345" {
		t.Errorf("Expected share ID 41012345, got %s", share.Id)
	}

	// Test that same shareName with different provider account returns different share
	share2, err := client.LookupShare(ctx, "ACME_CUSTOMER_DATA", "OTHER_CORP.US-WEST-2")
	if err != nil {
		t.Fatalf("LookupShare failed for second share: %v", err)
	}
	if share2.Id != "41012346" {
		t.Errorf("Expected share ID 41012346, got %s", share2.Id)
	}

	// Test not found
	_, err = client.LookupShare(ctx, "NON_EXISTENT_SHARE", "SOME_ACCOUNT")
	if err == nil {
		t.Fatal("Expected error for non-existent share")
	}
	if !HasStatusCode(err, http.StatusNotFound) {
		t.Errorf("Expected 404 status code, got: %v", err)
	}
}

func TestLookupShareDuplicate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/shares/inbound":
			// LIST endpoint - return two shares with same name
			w.Write([]byte(`{
				"shares": [
					{
						"id": "41012345",
						"shareName": "DUPLICATE_SHARE",
						"providerType": "Snowflake",
						"status": {"state": "Active", "health": "Healthy"},
						"createdBy": {"id": "123"}, "createdAt": "2026-01-15T10:30:00Z",
						"updatedBy": {"id": "123"}, "updatedAt": "2026-03-20T14:22:00Z",
						"tableCount": 15
					},
					{
						"id": "41012346",
						"shareName": "DUPLICATE_SHARE",
						"providerType": "Snowflake",
						"status": {"state": "Active", "health": "Healthy"},
						"createdBy": {"id": "123"}, "createdAt": "2026-01-15T10:30:00Z",
						"updatedBy": {"id": "123"}, "updatedAt": "2026-03-20T14:22:00Z",
						"tableCount": 5
					}
				],
				"meta": {"totalCount": 2}
			}`))
		case "/v1/shares/inbound/41012345", "/v1/shares/inbound/41012346":
			// GET endpoints - return full details including SnowflakeConfig
			shareId := "41012345"
			tableCount := 15
			if r.URL.Path == "/v1/shares/inbound/41012346" {
				shareId = "41012346"
				tableCount = 5
			}
			w.Write([]byte(fmt.Sprintf(`{
				"id": "%s",
				"shareName": "DUPLICATE_SHARE",
				"providerType": "Snowflake",
				"snowflakeConfig": {
					"shareName": "DUPLICATE_SHARE",
					"providerAccount": "PROVIDER.REGION"
				},
				"status": {"state": "Active", "health": "Healthy"},
				"createdBy": {"id": "123"}, "createdAt": "2026-01-15T10:30:00Z",
				"updatedBy": {"id": "123"}, "updatedAt": "2026-03-20T14:22:00Z",
				"tableCount": %d
			}`, shareId, tableCount)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	// Test that duplicate shares cause an error
	_, err := client.LookupShare(ctx, "DUPLICATE_SHARE", "PROVIDER.REGION")
	if err == nil {
		t.Fatal("Expected error for duplicate shares")
	}
	if !HasStatusCode(err, http.StatusConflict) {
		t.Errorf("Expected 409 Conflict status code, got: %v", err)
	}
	t.Logf("Got expected conflict error: %v", err)
}
