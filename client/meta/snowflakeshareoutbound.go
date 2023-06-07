package meta

import (
	"context"
	"fmt"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type snowflakeShareOutboundResponse interface {
	GetShare() SnowflakeShareOutbound
}

func snowflakeShareOutboundOrError(r snowflakeShareOutboundResponse, err error) (*SnowflakeShareOutbound, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetShare()
	return &result, nil
}

func (client *Client) GetSnowflakeShareOutbound(ctx context.Context, id string) (*SnowflakeShareOutbound, error) {
	resp, err := getSnowflakeShareOutbound(ctx, client.Gql, id)
	return snowflakeShareOutboundOrError(resp, err)
}

func (client *Client) LookupSnowflakeShareOutbound(ctx context.Context, name string, workspaceId string) (*SnowflakeShareOutbound, error) {
	resp, err := lookupSnowflakeShareOutbound(ctx, client.Gql, name, workspaceId)

	if err != nil {
		return nil, err
	}

	results := resp.Shares.Results

	if len(results) == 0 {
		return nil, fmt.Errorf("share not found with name %q in workspace %q", name, workspaceId)
	}

	return &results[0], nil
}

func (client *Client) CreateSnowflakeShareOutbound(ctx context.Context, workspaceId string, input *SnowflakeShareOutboundInput) (*SnowflakeShareOutbound, error) {
	resp, err := createSnowflakeShareOutbound(ctx, client.Gql, workspaceId, *input)
	return snowflakeShareOutboundOrError(resp, err)
}

func (client *Client) UpdateSnowflakeShareOutbound(ctx context.Context, id string, input *SnowflakeShareOutboundInput) (*SnowflakeShareOutbound, error) {
	resp, err := updateSnowflakeShareOutbound(ctx, client.Gql, id, *input)
	return snowflakeShareOutboundOrError(resp, err)
}

func (client *Client) DeleteSnowflakeShareOutbound(ctx context.Context, id string) error {
	resp, err := deleteSnowflakeShareOutbound(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (p *SnowflakeShareOutbound) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypeSnowflakeShareOutbound,
	}
}
