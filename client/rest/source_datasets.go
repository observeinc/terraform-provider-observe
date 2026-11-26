package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const sourceDatasetsPath = "/v1/source-datasets"

// SourceDatasetIcebergRef identifies a referenced object by id.
type SourceDatasetIcebergRef struct {
	Id string `json:"id"`
}

// SourceDatasetIceberg is the Iceberg backing of a source dataset as returned by the API.
type SourceDatasetIceberg struct {
	StorageIntegration *SourceDatasetIcebergRef `json:"storageIntegration"`
	ExternalWrite      bool                     `json:"externalWrite"`
}

// SourceDatasetIngest is the ingest backing of a source dataset as returned by the API.
type SourceDatasetIngest struct {
	Storage    string                `json:"storage"`
	SchemaType string                `json:"schemaType"`
	Iceberg    *SourceDatasetIceberg `json:"iceberg"`
}

// SourceDatasetDataSource describes how a source dataset is fed.
type SourceDatasetDataSource struct {
	Kind   string               `json:"kind"`
	Ingest *SourceDatasetIngest `json:"ingest"`
}

// SourceDatasetResource is a source dataset returned by the REST API.
type SourceDatasetResource struct {
	Id                         string                  `json:"id"`
	Label                      string                  `json:"label"`
	Description                *string                 `json:"description"`
	Icon                       *string                 `json:"icon"`
	ObjectTags                 map[string][]string     `json:"objectTags"`
	DataRetentionDays          *int64                  `json:"dataRetentionDays"`
	EffectiveDataRetentionDays int64                   `json:"effectiveDataRetentionDays"`
	DataSource                 SourceDatasetDataSource `json:"dataSource"`
}

// SourceDatasetIcebergInput is the Iceberg part of a create request.
type SourceDatasetIcebergInput struct {
	StorageIntegrationId string `json:"storageIntegrationId"`
	ExternalWrite        bool   `json:"externalWrite"`
}

// SourceDatasetIngestInput is the ingest part of a create request.
type SourceDatasetIngestInput struct {
	Storage    string                     `json:"storage"`
	SchemaType string                     `json:"schemaType"`
	Iceberg    *SourceDatasetIcebergInput `json:"iceberg,omitempty"`
}

// SourceDatasetDataSourceInput is the dataSource part of a create request.
type SourceDatasetDataSourceInput struct {
	Kind   string                   `json:"kind"`
	Ingest SourceDatasetIngestInput `json:"ingest"`
}

// SourceDatasetCreateRequest creates a source dataset.
type SourceDatasetCreateRequest struct {
	Label             string                       `json:"label"`
	Description       *string                      `json:"description,omitempty"`
	Icon              *string                      `json:"icon,omitempty"`
	ObjectTags        map[string][]string          `json:"objectTags,omitempty"`
	DataRetentionDays *int64                       `json:"dataRetentionDays,omitempty"`
	DataSource        SourceDatasetDataSourceInput `json:"dataSource"`
}

// SourceDatasetUpdateRequest is the source dataset merge-patch payload. For
// each nullable field, a nil value with its Specified flag set is serialized
// as JSON null and clears the field. ObjectTags is never sent as null: send an
// empty map to clear tags.
type SourceDatasetUpdateRequest struct {
	Label                      *string
	Description                *string
	DescriptionSpecified       bool
	Icon                       *string
	IconSpecified              bool
	ObjectTags                 *map[string][]string
	DataRetentionDays          *int64
	DataRetentionDaysSpecified bool
}

// MarshalJSON writes only the specified fields, using JSON null to clear them.
func (request SourceDatasetUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]interface{}{}
	if request.Label != nil {
		body["label"] = *request.Label
	}
	if request.DescriptionSpecified {
		body["description"] = request.Description
	}
	if request.IconSpecified {
		body["icon"] = request.Icon
	}
	if request.ObjectTags != nil {
		tags := *request.ObjectTags
		if tags == nil {
			tags = map[string][]string{}
		}
		body["objectTags"] = tags
	}
	if request.DataRetentionDaysSpecified {
		body["dataRetentionDays"] = request.DataRetentionDays
	}
	return json.Marshal(body)
}

func decodeSourceDataset(resp *http.Response) (*SourceDatasetResource, error) {
	defer resp.Body.Close()
	ds := &SourceDatasetResource{}
	if err := json.NewDecoder(resp.Body).Decode(ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func sourceDatasetPath(id string) string {
	return sourceDatasetsPath + "/" + url.PathEscape(id)
}

func (client *Client) CreateSourceDataset(ctx context.Context, req *SourceDatasetCreateRequest) (*SourceDatasetResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.requestWithContext(ctx, http.MethodPost, sourceDatasetsPath, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return decodeSourceDataset(resp)
}

func (client *Client) GetSourceDataset(ctx context.Context, id string) (*SourceDatasetResource, error) {
	resp, err := client.requestWithContext(ctx, http.MethodGet, sourceDatasetPath(id), "", nil)
	if err != nil {
		return nil, err
	}
	return decodeSourceDataset(resp)
}

func (client *Client) UpdateSourceDataset(ctx context.Context, id string, req *SourceDatasetUpdateRequest) (*SourceDatasetResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.requestWithContext(ctx, http.MethodPatch, sourceDatasetPath(id), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return decodeSourceDataset(resp)
}

func (client *Client) DeleteSourceDataset(ctx context.Context, id string, force bool) error {
	path := sourceDatasetPath(id)
	if force {
		path += "?force=true"
	}
	resp, err := client.requestWithContext(ctx, http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ListSourceDatasets returns the first page of source datasets matching the
// CEL filter (empty for all).
func (client *Client) ListSourceDatasets(ctx context.Context, filter string) ([]SourceDatasetResource, error) {
	path := sourceDatasetsPath
	if filter != "" {
		path += "?" + url.Values{"filter": []string{filter}}.Encode()
	}
	resp, err := client.requestWithContext(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		SourceDatasets []SourceDatasetResource `json:"sourceDatasets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.SourceDatasets, nil
}
