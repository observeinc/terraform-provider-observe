package meta

import (
	"context"
	"errors"
)

func (client *Client) CreateCorrelationTag(ctx context.Context, dataset, tag string, path LinkFieldInput) error {
	resp, err := addCorrelationTag(ctx, client.Gql, dataset, path, tag)
	if err != nil {
		return err
	}
	rs := resp.GetResultStatus().ResultStatus
	if !rs.Success {
		return errors.New(rs.ErrorMessage)
	}
	return nil
}

func (client *Client) DeleteCorrelationTag(ctx context.Context, dataset, tag string, path LinkFieldInput) error {
	resp, err := removeCorrelationTag(ctx, client.Gql, dataset, path, tag)
	if err != nil {
		return err
	}
	rs := resp.GetResultStatus().ResultStatus
	if !rs.Success {
		return errors.New(rs.ErrorMessage)
	}
	return nil
}

func (client *Client) IsCorrelationTagPresent(ctx context.Context, dataset, tag string, path LinkFieldInput) (bool, error) {
	resp, err := getDatasetCorrelationTags(ctx, client.Gql, dataset)
	if err != nil {
		return false, err
	}
	present := false
	for _, mapping := range resp.CorrelationTags.CorrelationTagMappings {
		if mapping.Tag == tag && mapping.Path.Column == path.Column && equalPtr(mapping.Path.Path, path.Path) {
			present = true
			break
		}
	}
	return present, nil
}

func equalPtr[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
