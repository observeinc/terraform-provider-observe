package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// IngestFilterResource is a drop filter as returned by the REST API.
type IngestFilterResource struct {
	Id            string     `json:"id"`
	Label         string     `json:"label"`
	SourceDataset DatasetRef `json:"sourceDataset"`
	Pipeline      string     `json:"pipeline"`
	DropRate      float64    `json:"dropRate"`
	Enabled       bool       `json:"enabled"`
}

// IngestFilterCreateRequest is the body for creating a drop filter.
//
// Enabled and DropRate must NOT be tagged omitempty: the server defaults them
// (enabled=true, dropRate=1.0) when the field is absent, so the zero values
// (false / 0.0) must be serialized explicitly to persist a disabled filter or
// a zero drop rate.
type IngestFilterCreateRequest struct {
	Label         string     `json:"label"`
	SourceDataset DatasetRef `json:"sourceDataset"`
	Pipeline      string     `json:"pipeline"`
	DropRate      float64    `json:"dropRate"`
	Enabled       bool       `json:"enabled"`
}

// IngestFilterUpdateRequest is the RFC 7396 merge-patch body for updating a
// drop filter. The source dataset is immutable server-side and therefore not
// included. Enabled and DropRate must NOT be omitempty (see the note on
// IngestFilterCreateRequest).
type IngestFilterUpdateRequest struct {
	Label    string  `json:"label"`
	Pipeline string  `json:"pipeline"`
	DropRate float64 `json:"dropRate"`
	Enabled  bool    `json:"enabled"`
}

func (f *IngestFilterResource) Oid() oid.OID {
	return oid.OID{
		Id:   f.Id,
		Type: oid.TypeIngestFilter,
	}
}

func (client *Client) decodeIngestFilterFromBody(resp *http.Response) (*IngestFilterResource, error) {
	defer resp.Body.Close()

	resource := &IngestFilterResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateIngestFilter(ctx context.Context, req *IngestFilterCreateRequest) (*IngestFilterResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post("/v1/ingest/filters", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestFilterFromBody(resp)
}

func (client *Client) GetIngestFilter(ctx context.Context, id string) (*IngestFilterResource, error) {
	resp, err := client.Get("/v1/ingest/filters/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestFilterFromBody(resp)
}

func (client *Client) UpdateIngestFilter(ctx context.Context, id string, req *IngestFilterUpdateRequest) (*IngestFilterResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/ingest/filters/"+url.PathEscape(id), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestFilterFromBody(resp)
}

func (client *Client) DeleteIngestFilter(ctx context.Context, id string) error {
	resp, err := client.Delete("/v1/ingest/filters/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (client *Client) ListIngestFilters(ctx context.Context) ([]IngestFilterResource, error) {
	resp, err := client.Get("/v1/ingest/filters?limit=1000")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		IngestFilters []IngestFilterResource `json:"ingestFilters"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.IngestFilters, nil
}
