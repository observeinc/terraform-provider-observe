package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateIngestFilter(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/filters" {
			t.Errorf("expected path /v1/ingest/filters, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		// The create endpoint returns 201, which must be accepted by the
		// client (the shared responseWrapper accepts any 2xx).
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "41030001",
			"label": "test-filter",
			"sourceDataset": {"id": "41007777"},
			"pipeline": "filter true",
			"dropRate": 0,
			"enabled": false
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	req := &IngestFilterCreateRequest{
		Label:         "test-filter",
		SourceDataset: DatasetRef{Id: "41007777"},
		Pipeline:      "filter true",
		DropRate:      0,
		Enabled:       false,
	}
	filter, err := client.CreateIngestFilter(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateIngestFilter failed: %v", err)
	}
	if filter.Id != "41030001" {
		t.Errorf("expected id 41030001, got %s", filter.Id)
	}
	if filter.SourceDataset.Id != "41007777" {
		t.Errorf("expected sourceDataset.id 41007777, got %s", filter.SourceDataset.Id)
	}
	if filter.Enabled {
		t.Errorf("expected enabled=false, got true")
	}

	// Regression guard: enabled=false and dropRate=0 must be serialized
	// explicitly (no omitempty), otherwise the server applies its defaults
	// (enabled=true, dropRate=1.0) and silently ignores the caller's intent.
	if _, ok := gotBody["enabled"]; !ok {
		t.Error("request body must include \"enabled\" even when false (omitempty regression)")
	}
	if v, ok := gotBody["enabled"].(bool); !ok || v {
		t.Errorf("expected enabled=false in request body, got %v", gotBody["enabled"])
	}
	if _, ok := gotBody["dropRate"]; !ok {
		t.Error("request body must include \"dropRate\" even when 0 (omitempty regression)")
	}
}

func TestGetIngestFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/filters/41030001" {
			t.Errorf("expected path /v1/ingest/filters/41030001, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "41030001",
			"label": "test-filter",
			"sourceDataset": {"id": "41007777"},
			"pipeline": "filter true",
			"dropRate": 0.5,
			"enabled": true
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	filter, err := client.GetIngestFilter(context.Background(), "41030001")
	if err != nil {
		t.Fatalf("GetIngestFilter failed: %v", err)
	}
	if got := string(filter.Oid().Type); got != "ingestfilter" {
		t.Errorf("expected oid type ingestfilter, got %s", got)
	}
	if filter.DropRate != 0.5 {
		t.Errorf("expected dropRate 0.5, got %v", filter.DropRate)
	}
}

func TestGetIngestFilterNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "No IngestFilter with id 41030001 is available."}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	_, err := client.GetIngestFilter(context.Background(), "41030001")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !HasStatusCode(err, http.StatusNotFound) {
		t.Errorf("expected 404 status code error, got %v", err)
	}
}

func TestUpdateIngestFilter(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/filters/41030001" {
			t.Errorf("expected path /v1/ingest/filters/41030001, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "41030001",
			"label": "renamed",
			"sourceDataset": {"id": "41007777"},
			"pipeline": "filter true",
			"dropRate": 1,
			"enabled": false
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	req := &IngestFilterUpdateRequest{
		Label:    "renamed",
		Pipeline: "filter true",
		DropRate: 1,
		Enabled:  false,
	}
	filter, err := client.UpdateIngestFilter(context.Background(), "41030001", req)
	if err != nil {
		t.Fatalf("UpdateIngestFilter failed: %v", err)
	}
	if filter.Label != "renamed" {
		t.Errorf("expected label renamed, got %s", filter.Label)
	}
	// The merge-patch body must not carry sourceDataset (immutable).
	if _, ok := gotBody["sourceDataset"]; ok {
		t.Error("update body must not include sourceDataset (source dataset is immutable)")
	}
	if _, ok := gotBody["enabled"]; !ok {
		t.Error("update body must include \"enabled\" even when false (omitempty regression)")
	}
}

func TestDeleteIngestFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/filters/41030001" {
			t.Errorf("expected path /v1/ingest/filters/41030001, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	if err := client.DeleteIngestFilter(context.Background(), "41030001"); err != nil {
		t.Fatalf("DeleteIngestFilter failed: %v", err)
	}
}

func TestListIngestFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/filters" {
			t.Errorf("expected path /v1/ingest/filters, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ingestFilters": [
				{"id": "41030001", "label": "a", "sourceDataset": {"id": "1"}, "pipeline": "filter true", "dropRate": 1, "enabled": true},
				{"id": "41030002", "label": "b", "sourceDataset": {"id": "2"}, "pipeline": "filter true", "dropRate": 1, "enabled": false}
			],
			"meta": {"totalCount": 2}
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	filters, err := client.ListIngestFilters(context.Background())
	if err != nil {
		t.Fatalf("ListIngestFilters failed: %v", err)
	}
	if len(filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(filters))
	}
	if filters[0].Label != "a" || filters[1].Label != "b" {
		t.Errorf("unexpected filter labels: %s, %s", filters[0].Label, filters[1].Label)
	}
}
