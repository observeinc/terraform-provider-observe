package meta

import (
	"context"
)

var (
	backendTaskResultFragment = `
	fragment taskResultFields on TaskResult {
		queryId
		stageId
		startTime
		endTime
		error
		resultCursor
		resultSchema {
			typedefDefinition
		}
	}`
)

// DatasetQueryOutput takes a simplified form: we use StageQueryInput instead of StageInput for now
func (c *Client) DatasetQueryOutput(ctx context.Context, query []*StageInput, params *QueryParams) ([]*TaskResult, error) {
	result, err := c.Run(ctx, backendTaskResultFragment+`
	query datasetQueryOutput($query: [StageInput!]!, $params: QueryParams!) {
		datasetQueryOutput(query: $query, params: $params) {
			...taskResultFields
		}
	}`, map[string]interface{}{
		"query":  query,
		"params": params,
	})

	if err != nil {
		return nil, err
	}
	var results []*TaskResult
	err = decodeStrict(getNested(result, "datasetQueryOutput"), &results)
	return results, err
}
