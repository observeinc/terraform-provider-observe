package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type preferredPathWithStatusResponse interface {
	GetPreferredPathWithStatus() PreferredPathWithStatus
}

func preferredPathWithStatusOrError(p preferredPathWithStatusResponse, err error) (*PreferredPathWithStatus, error) {
	if err != nil {
		return nil, err
	}
	result := p.GetPreferredPathWithStatus()
	return &result, nil
}

func (client *Client) CreatePreferredPath(ctx context.Context, workspaceId string, input *PreferredPathInput) (*PreferredPathWithStatus, error) {
	resp, err := createPreferredPath(ctx, client.Gql, workspaceId, *input)
	return preferredPathWithStatusOrError(resp, err)
}

func (client *Client) GetPreferredPath(ctx context.Context, id string) (*PreferredPathWithStatus, error) {
	resp, err := getPreferredPath(ctx, client.Gql, id)
	return preferredPathWithStatusOrError(resp, err)
}

func (client *Client) UpdatePreferredPath(ctx context.Context, id string, input *PreferredPathInput) (*PreferredPathWithStatus, error) {
	resp, err := updatePreferredPath(ctx, client.Gql, id, *input)
	return preferredPathWithStatusOrError(resp, err)
}

func (client *Client) DeletePreferredPath(ctx context.Context, id string) error {
	resp, err := deletePreferredPath(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (p *PreferredPath) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypePreferredPath,
	}
}
