package rest

import (
	"encoding/json"
	"testing"
)

func TestOpalVersionRequestSerialization(t *testing.T) {
	version := "v2026_09_30"
	tests := []struct {
		name    string
		request interface{}
	}{
		{
			name: "ingest route create",
			request: IngestRouteCreateRequest{
				Pipeline:      "filter true",
				DestinationId: "41000001",
				OpalVersion:   &version,
			},
		},
		{
			name: "ingest route update",
			request: IngestRouteUpdateRequest{
				OpalVersion: &version,
			},
		},
		{
			name: "ingest filter create",
			request: IngestFilterCreateRequest{
				OpalVersion: &version,
			},
		},
		{
			name: "ingest filter update",
			request: IngestFilterUpdateRequest{
				OpalVersion: &version,
			},
		},
		{
			name: "dataset query filter",
			request: DatasetQueryFilterDefinition{
				OpalVersion: &version,
			},
		},
		{
			name: "dashboard create",
			request: DashboardCreateInput{
				OpalVersion: &version,
			},
		},
		{
			name: "dashboard update",
			request: DashboardPatchInput{
				OpalVersion: &version,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}
			if got := decoded["opalVersion"]; got != version {
				t.Fatalf("opalVersion = %v, want %q (body %s)", got, version, body)
			}
		})
	}
}

func TestOpalVersionRequestOmittedWhenUnset(t *testing.T) {
	requests := []interface{}{
		IngestRouteCreateRequest{},
		IngestRouteUpdateRequest{},
		IngestFilterCreateRequest{},
		IngestFilterUpdateRequest{},
		DatasetQueryFilterDefinition{},
		DashboardCreateInput{},
		DashboardPatchInput{},
	}
	for _, request := range requests {
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("json.Marshal(%T) error: %v", request, err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(%T) error: %v", request, err)
		}
		if _, exists := decoded["opalVersion"]; exists {
			t.Fatalf("json.Marshal(%T) included unset opalVersion: %s", request, body)
		}
	}
}
