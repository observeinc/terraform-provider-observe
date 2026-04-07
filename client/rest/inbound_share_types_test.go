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

