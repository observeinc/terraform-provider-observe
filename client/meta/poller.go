package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type pollerResponse interface {
	GetPoller() Poller
}

func pollerOrError(p pollerResponse, err error) (*Poller, error) {
	if err != nil {
		return nil, err
	}
	result := p.GetPoller()
	return &result, nil
}

func (client *Client) CreatePoller(ctx context.Context, workspaceId string, input *PollerInput) (*Poller, error) {
	resp, err := createPoller(ctx, client.Gql, workspaceId, *input)
	return pollerOrError(resp, err)
}

func (client *Client) GetPoller(ctx context.Context, id string) (*Poller, error) {
	resp, err := getPoller(ctx, client.Gql, id)
	return pollerOrError(resp, err)
}

func (client *Client) UpdatePoller(ctx context.Context, id string, input *PollerInput) (*Poller, error) {
	resp, err := updatePoller(ctx, client.Gql, id, *input)
	return pollerOrError(resp, err)
}

func (client *Client) DeletePoller(ctx context.Context, id string) error {
	resp, err := deletePoller(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (p *Poller) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypePoller,
	}
}
