package rest

import (
	"encoding/json"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// Inbound Share API Types
// Based on OpenAPI spec at /code/openapi/sharein/sharein.yaml

// Meta represents pagination metadata
// Based on OpenAPI spec at /code/openapi/common/meta.yaml
type Meta struct {
	TotalCount int `json:"totalCount"`
}

// User represents a user reference in timestamps
type User struct {
	Id    string `json:"id"`
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

// DatasetRef represents a reference to a dataset. The Record field is
// populated when the server-side expand projection is honoured; consumers
// that need the human-readable metadata read Record.Label.
//
// NOTE: the custom UnmarshalJSON below accepts both the legacy flat
// {id, label} wire shape and the new nested {id, record} shape during
// the server-side migration. Once the migration completes, UnmarshalJSON
// can be deleted; the default struct unmarshaller is sufficient for
// nested-only wire traffic.
type DatasetRef struct {
	Id     string        `json:"id"`
	Record *DatasetBrief `json:"record,omitempty"`
}

// DatasetBrief carries the embedded dataset metadata that accompanies a
// dataset reference when the server-side expand projection is honoured.
type DatasetBrief struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// UnmarshalJSON accepts BOTH the legacy flat shape ({"id":..., "label":...})
// and the new nested shape ({"id":..., "record":{"label":...}}). The legacy
// shape is normalised into the nested representation so consumers only deal
// with one in-memory form. When both the top-level label and the nested
// record are present (possible during a transitional rollout), the nested
// form wins. A top-level JSON null decodes to DatasetRef{Id:"", Record:nil}
// via the default code path — no special case is needed.
//
// An empty Id ("") is not rejected at decode time: it is up to the caller
// (currently GetInboundShareTable in inbound_share_tables.go, which checks)
// to validate the invariant. Future consumers that read DatasetRef.Id
// directly inherit that responsibility.
func (d *DatasetRef) UnmarshalJSON(data []byte) error {
	var raw struct {
		Id     string        `json:"id"`
		Label  string        `json:"label"`
		Record *DatasetBrief `json:"record"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Id = raw.Id
	switch {
	case raw.Record != nil:
		d.Record = raw.Record
	case raw.Label != "":
		d.Record = &DatasetBrief{Label: raw.Label}
	default:
		d.Record = nil
	}
	return nil
}

// Share Types

type ShareStatus struct {
	State             string  `json:"state"`  // Pending, Creating, Active, Inactive, Error, Deleting
	Health            string  `json:"health"` // Healthy, Unhealthy, Unknown
	HealthMessage     *string `json:"healthMessage"`
	LastHealthCheckAt *string `json:"lastHealthCheckAt"`
}

// UnmarshalJSON accepts the legacy lastHealthCheck key from servers that
// predate the rename, preferring lastHealthCheckAt when both are present.
func (s *ShareStatus) UnmarshalJSON(data []byte) error {
	type plain ShareStatus
	var raw struct {
		plain
		Legacy *string `json:"lastHealthCheck"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = ShareStatus(raw.plain)
	if s.LastHealthCheckAt == nil {
		s.LastHealthCheckAt = raw.Legacy
	}
	return nil
}

type SnowflakeShareConfig struct {
	ShareName       string `json:"shareName"`
	ProviderAccount string `json:"providerAccount"`
}

type Share struct {
	Id              string                `json:"id"`
	ShareName       string                `json:"shareName"`
	ProviderType    string                `json:"providerType"` // "Snowflake"
	SnowflakeConfig *SnowflakeShareConfig `json:"snowflakeConfig,omitempty"`
	Status          ShareStatus           `json:"status"`
	CreatedBy       User                  `json:"createdBy"`
	CreatedAt       string                `json:"createdAt"`
	UpdatedBy       User                  `json:"updatedBy"`
	UpdatedAt       string                `json:"updatedAt"`
	TableCount      int                   `json:"tableCount"`
}

type ShareListResponse struct {
	Shares []Share `json:"shares"`
	Meta   Meta    `json:"meta"`
}

// Table Types

type ShareRef struct {
	Id        string `json:"id"`
	ShareName string `json:"shareName,omitempty"`
}

type ColumnDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type TableSchema struct {
	Columns []ColumnDefinition `json:"columns"`
}

type InboundShareTable struct {
	Id             string                  `json:"id"`
	Share          ShareRef                `json:"share"`
	FullTablePath  string                  `json:"fullTablePath"` // "schema/table"
	TableName      string                  `json:"tableName"`
	SchemaName     string                  `json:"schemaName"`
	TableType      string                  `json:"tableType"`     // "TABLE", "VIEW", etc.
	Status         string                  `json:"status"`        // Pending, Active, Inactive, Error
	SourceDataset  *DatasetRef             `json:"sourceDataset"` // Reference to the Observe dataset
	TableSchema    *TableSchema            `json:"tableSchema"`   // Snowflake table schema (columns, types)
	Description    string                  `json:"description,omitempty"`
	DatasetLabel   string                  `json:"datasetLabel,omitempty"` // Name/label of the Observe dataset
	DatasetKind    string                  `json:"datasetKind,omitempty"`  // Table, Event, Resource, Interval
	ValidFromField string                  `json:"validFromField,omitempty"`
	ValidToField   string                  `json:"validToField,omitempty"`
	FieldMapping   map[string]FieldMapping `json:"fieldMapping,omitempty"` // Schema mapping (field name -> type conversion)
	CreatedBy      User                    `json:"createdBy"`
	CreatedAt      string                  `json:"createdAt"`
	UpdatedBy      User                    `json:"updatedBy"`
	UpdatedAt      string                  `json:"updatedAt"`
}

type TableListResponse struct {
	Tables []InboundShareTable `json:"tables"`
	Meta   Meta                `json:"meta"`
}

// Dataset Types

type InboundShareDataset struct {
	Id        string `json:"id"`
	Label     string `json:"label"`
	Kind      string `json:"kind"` // Table, Resource, Event, Interval
	Source    string `json:"source"`
	CreatedBy User   `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	UpdatedBy User   `json:"updatedBy"`
	UpdatedAt string `json:"updatedAt"`
}

// Request/Response Types

type FieldMapping struct {
	Type       string `json:"type"`       // timestamp, duration, int64, string, etc.
	Conversion string `json:"conversion"` // Direct, MillisecondsToTimestamp, etc.
}

type TrackTableRequest struct {
	TableName      string                  `json:"tableName"`
	SchemaName     string                  `json:"schemaName"`
	DatasetLabel   string                  `json:"datasetLabel"`
	DatasetKind    string                  `json:"datasetKind"` // Table, Event, Resource, Interval
	ValidFromField *string                 `json:"validFromField,omitempty"`
	ValidToField   *string                 `json:"validToField,omitempty"`
	Description    *string                 `json:"description,omitempty"`
	SchemaMapping  map[string]FieldMapping `json:"schemaMapping,omitempty"`
}

type TrackTableResponse struct {
	Table   InboundShareTable   `json:"table"`
	Dataset InboundShareDataset `json:"dataset"`
}

// UnmarshalJSON accepts both track-table response shapes:
//   - the envelope {"table": {...}, "dataset": {...}}, and
//   - a bare table resource {"id": ..., "sourceDataset": {...}, ...}.
//
// For the bare form there is no dataset object, so Dataset.Id (and Label, when
// sourceDataset.record carries one) are taken from Table.SourceDataset. Kind is
// not available in that form; callers fill it from the dataset API.
func (r *TrackTableResponse) UnmarshalJSON(data []byte) error {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if _, ok := probe["table"]; ok {
		type plain TrackTableResponse
		var env plain
		if err := json.Unmarshal(data, &env); err != nil {
			return err
		}
		*r = TrackTableResponse(env)
		return nil
	}

	var table InboundShareTable
	if err := json.Unmarshal(data, &table); err != nil {
		return err
	}
	// Tracking always creates a dataset, so a bare table without an id or a
	// sourceDataset id is a malformed response, not an empty result. Failing
	// here keeps Terraform from dropping a table the server already tracked.
	if table.Id == "" {
		return fmt.Errorf("track table response has no table id")
	}
	if table.SourceDataset == nil || table.SourceDataset.Id == "" {
		return fmt.Errorf("track table response for table %s has no sourceDataset.id", table.Id)
	}
	*r = TrackTableResponse{Table: table}
	r.Dataset.Id = table.SourceDataset.Id
	if table.SourceDataset.Record != nil {
		r.Dataset.Label = table.SourceDataset.Record.Label
	}
	return nil
}

type UpdateTableRequest struct {
	Description    *string                 `json:"description,omitempty"`
	DatasetLabel   *string                 `json:"datasetLabel,omitempty"`
	ValidFromField *string                 `json:"validFromField,omitempty"`
	ValidToField   *string                 `json:"validToField,omitempty"`
	SchemaMapping  map[string]FieldMapping `json:"schemaMapping,omitempty"`
}

type UntrackedTable struct {
	TableName  string `json:"tableName"`
	SchemaName string `json:"schemaName"`
	TableType  string `json:"tableType"` // TABLE, VIEW, etc.
}

type UntrackedTableListResponse struct {
	Tables []UntrackedTable `json:"tables"`
	Meta   Meta             `json:"meta"`
}

// OID helpers

func (s *Share) Oid() oid.OID {
	return oid.OID{
		Id:   s.Id,
		Type: oid.TypeInboundShare,
	}
}

func (t *InboundShareTable) Oid() oid.OID {
	return oid.OID{
		Id:   t.Id,
		Type: oid.TypeInboundShareTable,
	}
}

func (d *InboundShareDataset) Oid() oid.OID {
	return oid.OID{
		Id:   d.Id,
		Type: oid.TypeDataset,
	}
}
