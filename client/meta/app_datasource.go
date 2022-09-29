package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type appDataSourceResponse interface {
	GetAppdatasource() AppDataSource
}

func appDataSourceOrError(a appDataSourceResponse, err error) (*AppDataSource, error) {
	if err != nil {
		return nil, err
	}
	result := a.GetAppdatasource()
	return &result, nil
}

func (client *Client) CreateAppDataSource(ctx context.Context, input *AppDataSourceInput) (*AppDataSource, error) {
	resp, err := createAppDataSource(ctx, client.Gql, *input)
	return appDataSourceOrError(resp, err)
}

func (client *Client) GetAppDataSource(ctx context.Context, id string) (*AppDataSource, error) {
	resp, err := getAppDataSource(ctx, client.Gql, id)
	return appDataSourceOrError(resp, err)
}

func (client *Client) UpdateAppDataSource(ctx context.Context, id string, input *AppDataSourceInput) (*AppDataSource, error) {
	resp, err := updateAppDataSource(ctx, client.Gql, id, *input)
	return appDataSourceOrError(resp, err)
}

func (client *Client) DeleteAppDataSource(ctx context.Context, id string) error {
	resp, err := deleteAppDataSource(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (a *AppDataSource) Oid() *oid.OID {
	return &oid.OID{
		Id:   a.Id,
		Type: oid.TypeAppDataSource,
	}
}
