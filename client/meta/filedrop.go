package meta

import (
	"context"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type filedropResponse interface {
	GetFiledrop() *Filedrop
}

func filedropOrError(f filedropResponse, err error) (*Filedrop, error) {
	if err != nil {
		return nil, err
	}
	result := f.GetFiledrop()
	return result, nil
}

func (client *Client) CreateFiledrop(ctx context.Context, workspaceId string, datastreamId string, input *FiledropInput) (*Filedrop, error) {
	resp, err := createFiledrop(ctx, client.Gql, workspaceId, datastreamId, *input)
	return filedropOrError(resp, err)
}

func (client *Client) GetFiledrop(ctx context.Context, id string) (*Filedrop, error) {
	resp, err := getFiledrop(ctx, client.Gql, id)
	return filedropOrError(resp, err)
}

func (client *Client) UpdateFiledrop(ctx context.Context, id string, input *FiledropInput) (*Filedrop, error) {
	resp, err := updateFiledrop(ctx, client.Gql, id, *input)
	return filedropOrError(resp, err)
}

func (client *Client) DeleteFiledrop(ctx context.Context, id string) error {
	resp, err := deleteFiledrop(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (f *Filedrop) Oid() *oid.OID {
	return &oid.OID{
		Id:   f.GetId(),
		Type: oid.TypeFiledrop,
	}
}
