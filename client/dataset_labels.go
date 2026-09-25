package client

import (
	"context"
	"fmt"
	"strconv"
)

// LookupDatasetLabels returns id→label for the given decimal dataset IDs that
// exist and are visible. Absent or hidden IDs are omitted. Empty input makes no
// request. Invalid IDs return an error.
func (c *Client) LookupDatasetLabels(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	parsed := make([]int64, len(ids))
	for i, id := range ids {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid dataset id %q", id)
		}
		parsed[i] = n
	}
	labels, err := c.Rest.LookupDatasetLabels(ctx, parsed)
	if err != nil {
		return nil, err
	}
	// Key by the caller's spelling, which may be non-canonical (e.g. "007").
	result := make(map[string]string, len(labels))
	for i, id := range ids {
		if label, ok := labels[parsed[i]]; ok {
			result[id] = label
		}
	}
	return result, nil
}
