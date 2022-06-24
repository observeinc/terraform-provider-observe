package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type datastreamResponse interface {
	GetDatastream() Datastream
}

func datastreamOrError(d datastreamResponse, err error) (*Datastream, error) {
	if err != nil {
		return nil, err
	}
	result := d.GetDatastream()
	return &result, nil
}

func (client *Client) CreateDatastream(ctx context.Context, workspaceId string, input *DatastreamInput) (*Datastream, error) {
	resp, err := createDatastream(ctx, client.Gql, workspaceId, *input)
	return datastreamOrError(resp, err)
}

func (client *Client) GetDatastream(ctx context.Context, id string) (*Datastream, error) {
	resp, err := getDatastream(ctx, client.Gql, id)
	return datastreamOrError(resp, err)
}

func (client *Client) UpdateDatastream(ctx context.Context, id string, input *DatastreamInput) (*Datastream, error) {
	resp, err := updateDatastream(ctx, client.Gql, id, *input)
	return datastreamOrError(resp, err)
}

func (client *Client) DeleteDatastream(ctx context.Context, id string) error {
	resp, err := deleteDatastream(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

// LookupDatastream retrieves datastream by name.
func (client *Client) LookupDatastream(ctx context.Context, workspaceId, name string) (*Datastream, error) {
	resp, err := lookupDatastream(ctx, client.Gql, workspaceId, name)
	return datastreamOrError(resp.Datastream, err)
}

func (d *Datastream) Oid() *oid.OID {
	return &oid.OID{
		Id:   d.Id,
		Type: oid.TypeDatastream,
	}
}
