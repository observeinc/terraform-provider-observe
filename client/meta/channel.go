package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type channelResponse interface {
	GetChannel() *Channel
}

func channelOrError(c channelResponse, err error) (*Channel, error) {
	if err != nil {
		return nil, err
	}
	return c.GetChannel(), nil
}

func (client *Client) CreateChannel(ctx context.Context, workspaceId string, input *ChannelInput) (*Channel, error) {
	resp, err := createChannel(ctx, client.Gql, workspaceId, *input)
	return channelOrError(resp, err)
}

func (client *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	resp, err := getChannel(ctx, client.Gql, id)
	return channelOrError(resp, err)
}

func (client *Client) UpdateChannel(ctx context.Context, id string, input *ChannelInput) (*Channel, error) {
	resp, err := updateChannel(ctx, client.Gql, id, *input)
	return channelOrError(resp, err)
}

func (client *Client) DeleteChannel(ctx context.Context, id string) error {
	resp, err := deleteChannel(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) SetMonitorsForChannel(ctx context.Context, id string, monitors []string) error {
	// endpoint does not accept null, set explicitly to empty list
	if monitors == nil {
		monitors = make([]string, 0)
	}
	resp, err := setMonitorsForChannel(ctx, client.Gql, id, monitors)
	return resultStatusError(resp, err)
}

func (c *Channel) Oid() *oid.OID {
	return &oid.OID{
		Id:   c.Id,
		Type: oid.TypeChannel,
	}
}
