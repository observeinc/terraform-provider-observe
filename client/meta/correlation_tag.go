package meta

import (
	"context"
)

func (client *Client) CreateCorrelationTag(ctx context.Context, dataset, tag string, path LinkFieldInput) error {
	resp, err := addCorrelationTag(ctx, client.Gql, dataset, path, tag)
	return resultStatusError(resp, err)
}

func (client *Client) DeleteCorrelationTag(ctx context.Context, dataset, tag string, path LinkFieldInput) error {
	resp, err := removeCorrelationTag(ctx, client.Gql, dataset, path, tag)
	return resultStatusError(resp, err)
}

func (client *Client) IsCorrelationTagPresent(ctx context.Context, dataset, tag string, path LinkFieldInput) (bool, error) {
	resp, err := getDatasetCorrelationTags(ctx, client.Gql, dataset)
	if err != nil {
		return false, err
	}
	present := false
	for _, mapping := range resp.CorrelationTags.CorrelationTagMappings {
		if mapping.Tag == tag && mapping.Path.Column == path.Column && pathsEqual(mapping.Path.Path, path.Path) {
			present = true
			break
		}
	}
	return present, nil
}

// pathsEqual treats nil and "" as equivalent representations of "no path": the backend
// echoes back "", while an import ID with no Path.Path field (e.g. from the documented
// import example, which omits it) unmarshals to nil.
func pathsEqual(a, b *string) bool {
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return deref(a) == deref(b)
}
