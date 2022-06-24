package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type deferredForeignKeyResponse interface {
	GetDeferredForeignKey() *DeferredForeignKey
}

func deferredForeignKeyOrError(d deferredForeignKeyResponse, err error) (*DeferredForeignKey, error) {
	if err != nil {
		return nil, err
	}
	return d.GetDeferredForeignKey(), nil
}

func (client *Client) CreateDeferredForeignKey(ctx context.Context, workspaceId string, input *DeferredForeignKeyInput) (*DeferredForeignKey, error) {
	resp, err := createDeferredForeignKey(ctx, client.Gql, workspaceId, *input)
	return deferredForeignKeyOrError(resp, err)
}

func (client *Client) GetDeferredForeignKey(ctx context.Context, id string) (*DeferredForeignKey, error) {
	resp, err := getDeferredForeignKey(ctx, client.Gql, id)
	return deferredForeignKeyOrError(resp, err)
}

func (client *Client) UpdateDeferredForeignKey(ctx context.Context, id string, input *DeferredForeignKeyInput) (*DeferredForeignKey, error) {
	resp, err := updateDeferredForeignKey(ctx, client.Gql, id, *input)
	return deferredForeignKeyOrError(resp, err)
}

func (client *Client) DeleteDeferredForeignKey(ctx context.Context, id string) error {
	resp, err := deleteDeferredForeignKey(ctx, client.Gql, id)
	return optionalResultStatusError(resp, err)
}

func (p *DeferredForeignKey) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypeLink,
	}
}
