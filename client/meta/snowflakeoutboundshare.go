package meta

import (
	"context"
	"fmt"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type snowflakeOutboundShareResponse interface {
	GetShare() SnowflakeOutboundShare
}

func snowflakeOutboundShareOrError(r snowflakeOutboundShareResponse, err error) (*SnowflakeOutboundShare, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetShare()
	return &result, nil
}

func (client *Client) GetSnowflakeOutboundShare(ctx context.Context, id string) (*SnowflakeOutboundShare, error) {
	resp, err := getSnowflakeOutboundShare(ctx, client.Gql, id)
	return snowflakeOutboundShareOrError(resp, err)
}

func (client *Client) LookupSnowflakeOutboundShare(ctx context.Context, name string, workspaceId string) (*SnowflakeOutboundShare, error) {
	resp, err := lookupSnowflakeOutboundShare(ctx, client.Gql, name, workspaceId)

	if err != nil {
		return nil, err
	}

	results := resp.Shares.Results

	if len(results) == 0 {
		return nil, fmt.Errorf("share not found with name %q in workspace %q", name, workspaceId)
	}

	return &results[0], nil
}

func (client *Client) CreateSnowflakeOutboundShare(ctx context.Context, workspaceId string, input *SnowflakeOutboundShareInput) (*SnowflakeOutboundShare, error) {
	resp, err := createSnowflakeOutboundShare(ctx, client.Gql, workspaceId, *input)
	return snowflakeOutboundShareOrError(resp, err)
}

func (client *Client) UpdateSnowflakeOutboundShare(ctx context.Context, id string, input *SnowflakeOutboundShareInput) (*SnowflakeOutboundShare, error) {
	resp, err := updateSnowflakeOutboundShare(ctx, client.Gql, id, *input)
	return snowflakeOutboundShareOrError(resp, err)
}

func (client *Client) DeleteSnowflakeOutboundShare(ctx context.Context, id string) error {
	resp, err := deleteSnowflakeOutboundShare(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (p *SnowflakeOutboundShare) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypeSnowflakeOutboundShare,
	}
}
