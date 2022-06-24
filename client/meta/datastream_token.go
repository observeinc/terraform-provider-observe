package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type datastreamTokenResponse interface {
	GetDatastreamToken() DatastreamToken
}

func datastreamTokenOrError(d datastreamTokenResponse, err error) (*DatastreamToken, error) {
	if err != nil {
		return nil, err
	}
	result := d.GetDatastreamToken()
	return &result, nil
}

func (client *Client) CreateDatastreamToken(ctx context.Context, workspaceId string, input *DatastreamTokenInput) (*DatastreamToken, error) {
	resp, err := createDatastreamToken(ctx, client.Gql, workspaceId, *input)
	return datastreamTokenOrError(resp, err)
}

func (client *Client) GetDatastreamToken(ctx context.Context, id string) (*DatastreamToken, error) {
	resp, err := getDatastreamToken(ctx, client.Gql, id)
	return datastreamTokenOrError(resp, err)
}

func (client *Client) UpdateDatastreamToken(ctx context.Context, id string, input *DatastreamTokenInput) (*DatastreamToken, error) {
	resp, err := updateDatastreamToken(ctx, client.Gql, id, *input)
	return datastreamTokenOrError(resp, err)
}

func (client *Client) DeleteDatastreamToken(ctx context.Context, id string) error {
	resp, err := deleteDatastreamToken(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (d *DatastreamToken) Oid() *oid.OID {
	return &oid.OID{
		Id:   d.Id,
		Type: oid.TypeDatastreamToken,
	}
}
