package meta

import "context"

func (c *Client) CreateIngestFilter(ctx context.Context, workspace string, input *IngestFilterInput) (*IngestFilter, error) {
	response, err := createIngestFilter(ctx, c.Gql, workspace, *input)
	return &response.IngestFilter, err
}

func (c *Client) GetIngestFilter(ctx context.Context, filterId string) (*IngestFilter, error) {
	response, err := getIngestFilter(ctx, c.Gql, filterId)
	return &response.IngestFilter, err
}

func (c *Client) UpdateIngestFilter(ctx context.Context, filterId string, input *IngestFilterInput) (*IngestFilter, error) {
	response, err := updateIngestFilter(ctx, c.Gql, filterId, *input)
	return &response.IngestFilter, err
}

func (c *Client) DeleteIngestFilter(ctx context.Context, filterId string) error {
	response, err := deleteIngestFilter(ctx, c.Gql, filterId)
	return resultStatusError(response, err)
}
