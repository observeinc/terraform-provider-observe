package rest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	if table.SourceDataset.Label != "Updated Label" {
		t.Errorf("Expected dataset label 'Updated Label', got %s", table.SourceDataset.Label)
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
