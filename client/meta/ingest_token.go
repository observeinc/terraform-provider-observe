package meta

import (
	"context"
)

func (c *Client) CreateIngestToken(
	ctx context.Context,
	workspace string,
	input IngestTokenInput,
) (*IngestToken, error) {
	response, err := createIngestToken(ctx, c.Gql, workspace, input)
	return &response.IngestToken, err
}

func (c *Client) GetIngestToken(
	ctx context.Context,
	id string,
) (*IngestToken, error) {
	response, err := getIngestToken(ctx, c.Gql, id)
	return &response.IngestToken, err
}

func (c *Client) UpdateIngestToken(
	ctx context.Context,
	id string,
	input IngestTokenInput,
) (*IngestToken, error) {
	response, err := updateIngestToken(ctx, c.Gql, id, input)
	return &response.IngestToken, err
}

func (c *Client) DeleteIngestToken(
	ctx context.Context,
	id string,
) error {
	response, err := deleteIngestToken(ctx, c.Gql, id)
	return resultStatusError(response, err)
}
