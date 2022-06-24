package meta

import (
	"context"
)

// GetDatasetQueryOutput takes a simplified form: we use StageQueryInput instead of StageInput for now
func (client *Client) DatasetQueryOutput(ctx context.Context, query []*StageInput, params *QueryParams) ([]*TaskResult, error) {
	resp, err := getDatasetQueryOutput(ctx, client.Gql, query, *params)
	if err != nil {
		return nil, err
	}
	return resp.TaskResult, nil
}
