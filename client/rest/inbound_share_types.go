package rest

import "github.com/observeinc/terraform-provider-observe/client/oid"

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

// DatasetRef represents a reference to a dataset
type DatasetRef struct {
	Id    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// Share Types

type ShareStatus struct {
	State           string  `json:"state"`  // Pending, Creating, Active, Inactive, Error, Deleting
	Health          string  `json:"health"` // Healthy, Unhealthy, Unknown
	HealthMessage   *string `json:"healthMessage"`
	LastHealthCheck *string `json:"lastHealthCheck"`
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
	Id            string       `json:"id"`
	Share         ShareRef     `json:"share"`
	FullTablePath string       `json:"fullTablePath"` // "schema/table"
	TableName     string       `json:"tableName"`
	SchemaName    string       `json:"schemaName"`
	TableType     string       `json:"tableType"` // "TABLE", "VIEW", etc.
	Status        string       `json:"status"`    // Pending, Active, Inactive, Error
	SourceDataset *DatasetRef  `json:"sourceDataset"`
	TableSchema   *TableSchema `json:"tableSchema"`
	CreatedBy     User         `json:"createdBy"`
	CreatedAt     string       `json:"createdAt"`
	UpdatedBy     User         `json:"updatedBy"`
	UpdatedAt     string       `json:"updatedAt"`
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
