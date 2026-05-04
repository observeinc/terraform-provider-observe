package rest

import (
	"encoding/json"
	"testing"
)

func TestShareJSONMarshaling(t *testing.T) {
	shareJSON := `{
		"id": "41012345",
		"shareName": "ACME_CUSTOMER_DATA",
		"providerType": "Snowflake",
		"snowflakeConfig": {
			"shareName": "CUSTOMER_SHARE_PROD",
			"providerAccount": "ACME_CORP.US-EAST-1"
		},
		"status": {
			"state": "Active",
			"health": "Healthy",
			"healthMessage": null,
			"lastHealthCheck": "2026-03-26T10:00:00Z"
		},
		"createdBy": {
			"id": "123",
			"email": "user@example.com"
		},
		"createdAt": "2026-01-15T10:30:00Z",
		"updatedBy": {
			"id": "123",
			"email": "user@example.com"
		},
		"updatedAt": "2026-03-20T14:22:00Z",
		"tableCount": 15
	}`

	var share Share
	if err := json.Unmarshal([]byte(shareJSON), &share); err != nil {
		t.Fatalf("Failed to unmarshal share: %v", err)
	}

	if share.Id != "41012345" {
		t.Errorf("Expected Id to be 41012345, got %s", share.Id)
	}
	if share.ShareName != "ACME_CUSTOMER_DATA" {
		t.Errorf("Expected ShareName to be ACME_CUSTOMER_DATA, got %s", share.ShareName)
	}
	if share.Status.State != "Active" {
		t.Errorf("Expected Status.State to be Active, got %s", share.Status.State)
	}
	if share.TableCount != 15 {
		t.Errorf("Expected TableCount to be 15, got %d", share.TableCount)
	}

	// Test marshaling back
	data, err := json.Marshal(&share)
	if err != nil {
		t.Fatalf("Failed to marshal share: %v", err)
	}
	if len(data) == 0 {
		t.Error("Marshaled data is empty")
	}
}

func TestTrackTableRequestMarshaling(t *testing.T) {
	desc := "Test description"
	validFrom := "event_timestamp"

	req := TrackTableRequest{
		TableName:      "CUSTOMER_EVENTS",
		SchemaName:     "PUBLIC",
		DatasetLabel:   "Customer Events",
		DatasetKind:    "Event",
		ValidFromField: &validFrom,
		Description:    &desc,
		SchemaMapping: map[string]FieldMapping{
			"event_timestamp": {
				Type:       "timestamp",
				Conversion: "MillisecondsToTimestamp",
			},
		},
	}

	data, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var unmarshaled TrackTableRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.TableName != "CUSTOMER_EVENTS" {
		t.Errorf("Expected TableName to be CUSTOMER_EVENTS, got %s", unmarshaled.TableName)
	}
	if unmarshaled.DatasetKind != "Event" {
		t.Errorf("Expected DatasetKind to be Event, got %s", unmarshaled.DatasetKind)
	}
	if unmarshaled.ValidFromField == nil || *unmarshaled.ValidFromField != "event_timestamp" {
		t.Error("ValidFromField not correctly unmarshaled")
	}
}

func TestTrackTableResponseMarshaling(t *testing.T) {
	responseJSON := `{
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
	}`

	var response TrackTableResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Table.Id != "41056789" {
		t.Errorf("Expected Table.Id to be 41056789, got %s", response.Table.Id)
	}
	if response.Dataset.Id != "41067890" {
		t.Errorf("Expected Dataset.Id to be 41067890, got %s", response.Dataset.Id)
	}
	if response.Dataset.Kind != "Event" {
		t.Errorf("Expected Dataset.Kind to be Event, got %s", response.Dataset.Kind)
	}
}

func TestDatasetRef_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ref DatasetRef)
	}{
		{
			name: "NestedShape",
			input: `{
				"id": "41067890",
				"record": {
					"label": "Customer Events",
					"description": "Snowflake share for customer events",
					"icon": "icons/event"
				}
			}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "41067890" {
					t.Errorf("expected Id 41067890, got %q", ref.Id)
				}
				if ref.Record == nil {
					t.Fatal("expected Record to be non-nil")
				}
				if ref.Record.Label != "Customer Events" {
					t.Errorf("expected Record.Label %q, got %q", "Customer Events", ref.Record.Label)
				}
				if ref.Record.Description != "Snowflake share for customer events" {
					t.Errorf("expected Record.Description %q, got %q", "Snowflake share for customer events", ref.Record.Description)
				}
				if ref.Record.Icon != "icons/event" {
					t.Errorf("expected Record.Icon %q, got %q", "icons/event", ref.Record.Icon)
				}
			},
		},
		{
			name:  "FlatShape",
			input: `{"id":"41067890","label":"Customer Events"}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "41067890" {
					t.Errorf("expected Id 41067890, got %q", ref.Id)
				}
				if ref.Record == nil {
					t.Fatal("expected Record to be synthesised for flat shape")
				}
				if ref.Record.Label != "Customer Events" {
					t.Errorf("expected Record.Label %q, got %q", "Customer Events", ref.Record.Label)
				}
				if ref.Record.Description != "" {
					t.Errorf("expected Record.Description empty for flat shape, got %q", ref.Record.Description)
				}
				if ref.Record.Icon != "" {
					t.Errorf("expected Record.Icon empty for flat shape, got %q", ref.Record.Icon)
				}
			},
		},
		{
			name:  "IdOnly",
			input: `{"id":"41067890"}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "41067890" {
					t.Errorf("expected Id 41067890, got %q", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected Record to be nil for id-only response, got %+v", ref.Record)
				}
			},
		},
		{
			// During a transitional rollout the server could emit both the
			// top-level label and the nested record. The nested form must win.
			name: "BothPresent_NestedWins",
			input: `{
				"id": "41067890",
				"label": "old flat label",
				"record": {"label": "new nested label"}
			}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Record == nil {
					t.Fatal("expected Record to be populated from nested shape")
				}
				if ref.Record.Label != "new nested label" {
					t.Errorf("expected nested label to win, got %q", ref.Record.Label)
				}
			},
		},
		{
			// Explicit empty flat label is treated as id-only: no synthesised
			// blank brief. Deliberate; goes away when UnmarshalJSON is deleted
			// post-migration — future refactors should not "fix" it to
			// synthesise an empty brief.
			name:  "FlatLabelEmpty",
			input: `{"id":"41067890","label":""}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Record != nil {
					t.Errorf("expected Record nil for empty flat label, got %+v", ref.Record)
				}
			},
		},
		{
			name:  "RecordExplicitNull",
			input: `{"id":"x","record":null}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "x" {
					t.Errorf("expected Id %q, got %q", "x", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected Record nil for explicit null, got %+v", ref.Record)
				}
			},
		},
		{
			name:  "LabelExplicitNull",
			input: `{"id":"x","label":null}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "x" {
					t.Errorf("expected Id %q, got %q", "x", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected Record nil for null label, got %+v", ref.Record)
				}
			},
		},
		{
			name:    "MalformedJSON",
			input:   `{`,
			wantErr: true,
		},
		{
			name:    "RecordArrayType",
			input:   `{"id":"x","record":[]}`,
			wantErr: true,
		},
		{
			// record arriving as a JSON string (rather than an object) must
			// surface an error, not silently fall through to the flat or
			// id-only branch. On decode error the receiver is left zero; a
			// future refactor that assigns d.Id or d.Record before the shape
			// check would fail the zero-state assertions below.
			name:    "RecordWrongType",
			input:   `{"id":"41067890","record":"not an object"}`,
			wantErr: true,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "" {
					t.Errorf("expected ref.Id to be empty on decode error, got %q", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected ref.Record to be nil on decode error, got %+v", ref.Record)
				}
			},
		},
		{
			// encoding/json takes last-wins on duplicate keys; pin that so a
			// future refactor that switches to a strict streaming decoder
			// surfaces the change instead of silently flipping semantics.
			name:  "DuplicateIdKeys_LastWins",
			input: `{"id":"a","id":"b"}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "b" {
					t.Errorf("expected last-wins Id %q, got %q", "b", ref.Id)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ref DatasetRef
			err := json.Unmarshal([]byte(tc.input), &ref)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, ref)
			}
		})
	}
}

func TestTrackTableResponseMarshaling_NestedShape(t *testing.T) {
	// Sibling of TestTrackTableResponseMarshaling (which exercises the flat
	// wire shape). This fixture uses the post-migration nested form.
	responseJSON := `{
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
	}`

	var response TrackTableResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Table.Id != "41056789" {
		t.Errorf("Expected Table.Id 41056789, got %s", response.Table.Id)
	}
	if response.Table.SourceDataset == nil {
		t.Fatal("Expected Table.SourceDataset to be non-nil")
	}
	if response.Table.SourceDataset.Record == nil {
		t.Fatal("Expected Table.SourceDataset.Record to be populated by nested shape")
	}
	if response.Table.SourceDataset.Record.Label != "Customer Events" {
		t.Errorf("Expected nested label 'Customer Events', got %s", response.Table.SourceDataset.Record.Label)
	}
	if response.Table.SourceDataset.Record.Description != "Snowflake share for customer events" {
		t.Errorf("Expected nested description, got %q", response.Table.SourceDataset.Record.Description)
	}
	if response.Table.SourceDataset.Record.Icon != "icons/event" {
		t.Errorf("Expected nested icon 'icons/event', got %q", response.Table.SourceDataset.Record.Icon)
	}
	if response.Dataset.Id != "41067890" {
		t.Errorf("Expected Dataset.Id 41067890, got %s", response.Dataset.Id)
	}
}
