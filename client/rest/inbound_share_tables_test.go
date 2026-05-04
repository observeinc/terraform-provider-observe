package rest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrackTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345/tables" {
			t.Errorf("Expected path /v1/shares/inbound/41012345/tables, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Read and validate request body
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("Expected non-empty request body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"table": {
				"id": "41056789",
				"share": {
					"id": "41012345",
					"shareName": "ACME_CUSTOMER_DATA"
				},
				"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
				"tableName": "CUSTOMER_EVENTS",
				"schemaName": "PUBLIC",
				"tableType": "TABLE",
				"status": "Active",
				"sourceDataset": {
					"id": "41067890",
					"label": "Customer Events"
				},
				"createdBy": {"id": "123"},
				"createdAt": "2026-03-26T10:00:00Z",
				"updatedBy": {"id": "123"},
				"updatedAt": "2026-03-26T10:00:00Z"
			},
			"dataset": {
				"id": "41067890",
				"label": "Customer Events",
				"kind": "Event",
				"source": "sharein_table_41056789",
				"createdBy": {"id": "123"},
				"createdAt": "2026-03-26T10:00:00Z",
				"updatedBy": {"id": "123"},
				"updatedAt": "2026-03-26T10:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	validFrom := "event_timestamp"
	req := &TrackTableRequest{
		TableName:      "CUSTOMER_EVENTS",
		SchemaName:     "PUBLIC",
		DatasetLabel:   "Customer Events",
		DatasetKind:    "Event",
		ValidFromField: &validFrom,
	}

	result, err := client.TrackTable(ctx, "41012345", req)
	if err != nil {
		t.Fatalf("TrackTable failed: %v", err)
	}
	if result.Table.Id != "41056789" {
		t.Errorf("Expected table ID 41056789, got %s", result.Table.Id)
	}
	if result.Dataset.Id != "41067890" {
		t.Errorf("Expected dataset ID 41067890, got %s", result.Dataset.Id)
	}
	if result.Dataset.Kind != "Event" {
		t.Errorf("Expected dataset kind Event, got %s", result.Dataset.Kind)
	}
}

func TestGetInboundShareTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345/tables/41056789" {
			t.Errorf("Expected path /v1/shares/inbound/41012345/tables/41056789, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "41056789",
			"share": {
				"id": "41012345",
				"shareName": "ACME_CUSTOMER_DATA"
			},
			"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
			"tableName": "CUSTOMER_EVENTS",
			"schemaName": "PUBLIC",
			"tableType": "TABLE",
			"status": "Active",
			"sourceDataset": {
				"id": "41067890",
				"label": "Customer Events"
			},
			"createdBy": {"id": "123"},
			"createdAt": "2026-03-26T10:00:00Z",
			"updatedBy": {"id": "123"},
			"updatedAt": "2026-03-26T10:00:00Z"
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	result, err := client.GetInboundShareTable(ctx, "41012345", "41056789")
	if err != nil {
		t.Fatalf("GetInboundShareTable failed: %v", err)
	}
	if result.Table.Id != "41056789" {
		t.Errorf("Expected table ID 41056789, got %s", result.Table.Id)
	}
	if result.Dataset.Id != "41067890" {
		t.Errorf("Expected dataset ID 41067890, got %s", result.Dataset.Id)
	}
}

func TestUpdateInboundShareTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345/tables/41056789" {
			t.Errorf("Expected path /v1/shares/inbound/41012345/tables/41056789, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "41056789",
			"share": {
				"id": "41012345"
			},
			"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
			"tableName": "CUSTOMER_EVENTS",
			"schemaName": "PUBLIC",
			"tableType": "TABLE",
			"status": "Active",
			"sourceDataset": {
				"id": "41067890",
				"label": "Updated Label"
			},
			"createdBy": {"id": "123"},
			"createdAt": "2026-03-26T10:00:00Z",
			"updatedBy": {"id": "123"},
			"updatedAt": "2026-03-26T10:05:00Z"
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	newLabel := "Updated Label"
	req := &UpdateTableRequest{
		DatasetLabel: &newLabel,
	}

	table, err := client.UpdateInboundShareTable(ctx, "41012345", "41056789", req)
	if err != nil {
		t.Fatalf("UpdateInboundShareTable failed: %v", err)
	}
	if table.Id != "41056789" {
		t.Errorf("Expected table ID 41056789, got %s", table.Id)
	}
	if table.SourceDataset.Record == nil {
		t.Fatalf("Expected SourceDataset.Record to be non-nil")
	}
	if table.SourceDataset.Record.Label != "Updated Label" {
		t.Errorf("Expected dataset label 'Updated Label', got %s", table.SourceDataset.Record.Label)
	}
}

func TestDeleteInboundShareTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/shares/inbound/41012345/tables/41056789" {
			t.Errorf("Expected path /v1/shares/inbound/41012345/tables/41056789, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	err := client.DeleteInboundShareTable(ctx, "41012345", "41056789")
	if err != nil {
		t.Fatalf("DeleteInboundShareTable failed: %v", err)
	}
}

// newGetInboundShareTableServer returns an httptest server that asserts the request
// matches the shareId/tableId GET contract and replies 200 OK with
// responseJSON. Helper shared by the nested-shape and id-only read-path
// tests. Always responds 200 — not suitable for testing non-success status
// codes; extend when that need arises.
func newGetInboundShareTableServer(t *testing.T, shareId, tableId, responseJSON string) *httptest.Server {
	t.Helper()
	wantPath := "/v1/shares/inbound/" + shareId + "/tables/" + tableId
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("Expected path %s, got %s", wantPath, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
}

func TestGetInboundShareTable_NestedShape(t *testing.T) {
	// Sibling of TestGetInboundShareTable (flat wire shape). Same endpoint,
	// same client call, nested JSON over the wire — result must be identical.
	server := newGetInboundShareTableServer(t, "41012345", "41056789", `{
		"id": "41056789",
		"share": {
			"id": "41012345",
			"shareName": "ACME_CUSTOMER_DATA"
		},
		"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
		"tableName": "CUSTOMER_EVENTS",
		"schemaName": "PUBLIC",
		"tableType": "TABLE",
		"status": "Active",
		"sourceDataset": {
			"id": "41067890",
			"record": {
				"label": "Customer Events",
				"description": "Snowflake share for customer events",
				"icon": "icons/event"
			}
		},
		"createdBy": {"id": "123"},
		"createdAt": "2026-03-26T10:00:00Z",
		"updatedBy": {"id": "123"},
		"updatedAt": "2026-03-26T10:00:00Z"
	}`)
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	result, err := client.GetInboundShareTable(ctx, "41012345", "41056789")
	if err != nil {
		t.Fatalf("GetInboundShareTable failed: %v", err)
	}
	if result.Table.Id != "41056789" {
		t.Errorf("Expected table ID 41056789, got %s", result.Table.Id)
	}
	if result.Dataset.Id != "41067890" {
		t.Errorf("Expected dataset ID 41067890, got %s", result.Dataset.Id)
	}
	if result.Dataset.Label != "Customer Events" {
		t.Errorf("Expected dataset label 'Customer Events', got %s", result.Dataset.Label)
	}
}

func TestGetInboundShareTable_IdOnlyResponse(t *testing.T) {
	// If the sharein endpoint were ever to return an id-only sourceDataset
	// (no flat label, no nested record), GetInboundShareTable must refuse
	// the response with a clear error. This documents the read-path
	// contract: id-only is not sufficient for a Terraform read.
	server := newGetInboundShareTableServer(t, "41012345", "41056789", `{
		"id": "41056789",
		"share": {"id": "41012345"},
		"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
		"tableName": "CUSTOMER_EVENTS",
		"schemaName": "PUBLIC",
		"tableType": "TABLE",
		"status": "Active",
		"sourceDataset": {"id": "41067890"},
		"createdBy": {"id": "123"},
		"createdAt": "2026-03-26T10:00:00Z",
		"updatedBy": {"id": "123"},
		"updatedAt": "2026-03-26T10:00:00Z"
	}`)
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	_, err := client.GetInboundShareTable(ctx, "41012345", "41056789")
	if err == nil {
		t.Fatal("expected error for id-only sourceDataset response, got nil")
	}
	// Assert against the full Record-nil prefix (not just "sourceDataset.record")
	// so a future reorder that trips the Record.Label branch on a pathological
	// payload would fail this test instead of silently passing the wrong
	// branch.
	if !strings.Contains(err.Error(), "missing sourceDataset.record for table") {
		t.Errorf("expected error to mention 'missing sourceDataset.record for table', got %v", err)
	}
}
