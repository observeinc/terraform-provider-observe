package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// TrackTable tracks a table from the share and creates an associated Observe dataset
func (c *Client) TrackTable(ctx context.Context, shareId string, req *TrackTableRequest) (*TrackTableResponse, error) {
	path := fmt.Sprintf("/v1/shares/inbound/%s/tables", url.PathEscape(shareId))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.Post(path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TrackTableResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode track table response: %w", err)
	}

	return &result, nil
}

// GetInboundShareTable retrieves details for a specific tracked table
func (c *Client) GetInboundShareTable(ctx context.Context, shareId, tableId string) (*TrackTableResponse, error) {
	path := fmt.Sprintf("/v1/shares/inbound/%s/tables/%s",
		url.PathEscape(shareId), url.PathEscape(tableId))

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// The API returns a TableResource, but we want to return the same format as TrackTable
	// which includes both table and dataset info
	var table InboundShareTable
	if err := json.NewDecoder(resp.Body).Decode(&table); err != nil {
		return nil, fmt.Errorf("failed to decode table response: %w", err)
	}

	// For now, we'll return a TrackTableResponse with the table
	// The dataset info is embedded in table.SourceDataset
	result := &TrackTableResponse{
		Table: table,
	}

	// If there's a source dataset, populate it
	if table.SourceDataset != nil {
		result.Dataset = InboundShareDataset{
			Id:    table.SourceDataset.Id,
			Label: table.SourceDataset.Label,
		}
	}

	return result, nil
}

// UpdateInboundShareTable updates table metadata and dataset configuration
func (c *Client) UpdateInboundShareTable(ctx context.Context, shareId, tableId string, req *UpdateTableRequest) (*InboundShareTable, error) {
	path := fmt.Sprintf("/v1/shares/inbound/%s/tables/%s",
		url.PathEscape(shareId), url.PathEscape(tableId))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.Patch(path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var table InboundShareTable
	if err := json.NewDecoder(resp.Body).Decode(&table); err != nil {
		return nil, fmt.Errorf("failed to decode table response: %w", err)
	}

	return &table, nil
}

// DeleteInboundShareTable untracks a table from the share
func (c *Client) DeleteInboundShareTable(ctx context.Context, shareId, tableId string) error {
	path := fmt.Sprintf("/v1/shares/inbound/%s/tables/%s",
		url.PathEscape(shareId), url.PathEscape(tableId))

	resp, err := c.Delete(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
