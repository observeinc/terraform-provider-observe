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

type DatasetQueryFilterResource struct {
	Id          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Filter      string   `json:"filter"`
	Disabled    bool     `json:"disabled"`
	Errors      []string `json:"errors,omitempty"`
}

type DatasetQueryFilterDefinition struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Filter      string `json:"filter"`
	Disabled    bool   `json:"disabled"`
}

func (dqf *DatasetQueryFilterResource) Oid() oid.OID {
	return oid.OID{
		Id:   dqf.Id,
		Type: oid.TypeDatasetQueryFilter,
	}
}

func (client *Client) decodeDatasetQueryFilterFromBody(resp *http.Response) (*DatasetQueryFilterResource, error) {
	defer resp.Body.Close()

	resource := &DatasetQueryFilterResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateDatasetQueryFilter(ctx context.Context, datasetId string, req *DatasetQueryFilterDefinition) (*DatasetQueryFilterResource, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post(fmt.Sprintf("/v1/datasets/%s/query-filters", url.PathEscape(datasetId)), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return client.decodeDatasetQueryFilterFromBody(resp)
}

func (client *Client) GetDatasetQueryFilter(ctx context.Context, datasetId, id string) (*DatasetQueryFilterResource, error) {
	resp, err := client.Get(fmt.Sprintf("/v1/datasets/%s/query-filters/%s", url.PathEscape(datasetId), url.PathEscape(id)))
	if err != nil {
		return nil, err
	}
	return client.decodeDatasetQueryFilterFromBody(resp)
}

func (client *Client) UpdateDatasetQueryFilter(ctx context.Context, datasetId, id string, req *DatasetQueryFilterDefinition) (*DatasetQueryFilterResource, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch(fmt.Sprintf("/v1/datasets/%s/query-filters/%s", url.PathEscape(datasetId), url.PathEscape(id)), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return client.decodeDatasetQueryFilterFromBody(resp)
}

func (client *Client) DeleteDatasetQueryFilter(ctx context.Context, datasetId, id string) error {
	resp, err := client.Delete(fmt.Sprintf("/v1/datasets/%s/query-filters/%s", url.PathEscape(datasetId), url.PathEscape(id)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
