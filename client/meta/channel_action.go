package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type channelActionResponse interface {
	GetChannelAction() *ChannelAction
}

func channelActionOrError(c channelActionResponse, err error) (*ChannelAction, error) {
	if err != nil {
		return nil, err
	}
	return c.GetChannelAction(), nil
}

func (client *Client) CreateChannelAction(ctx context.Context, workspaceId string, input *ActionInput) (*ChannelAction, error) {
	resp, err := createChannelAction(ctx, client.Gql, workspaceId, *input)
	return channelActionOrError(resp, err)
}

func (client *Client) GetChannelAction(ctx context.Context, id string) (*ChannelAction, error) {
	resp, err := getChannelAction(ctx, client.Gql, id)
	return channelActionOrError(resp, err)
}

func (client *Client) UpdateChannelAction(ctx context.Context, id string, input *ActionInput) (*ChannelAction, error) {
	resp, err := updateChannelAction(ctx, client.Gql, id, *input)
	return channelActionOrError(resp, err)
}

func (client *Client) DeleteChannelAction(ctx context.Context, id string) error {
	resp, err := deleteChannelAction(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) SetChannelsForChannelAction(ctx context.Context, id string, channels []string) error {
	// endpoint does not accept null, set explicitly to empty list
	if channels == nil {
		channels = make([]string, 0)
	}
	resp, err := setChannelsForChannelAction(ctx, client.Gql, id, channels)
	return resultStatusError(resp, err)
}

func ChannelActionOid(c ChannelAction) *oid.OID {
	return &oid.OID{
		Id:   c.GetId(),
		Type: oid.TypeChannelAction,
	}
}
