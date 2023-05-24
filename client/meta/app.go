package meta

import (
	"context"
	"errors"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type appResponse interface {
	GetApp() App
}

func appOrError(a appResponse, err error) (*App, error) {
	if err != nil {
		return nil, err
	}
	result := a.GetApp()
	return &result, nil
}

func (client *Client) CreateApp(ctx context.Context, workspaceId string, input *AppInput) (*App, error) {
	resp, err := createApp(ctx, client.Gql, workspaceId, *input)
	return appOrError(resp, err)
}

func (client *Client) GetApp(ctx context.Context, id string) (*App, error) {
	resp, err := getApp(ctx, client.Gql, id)
	return appOrError(resp, err)
}

func (client *Client) UpdateApp(ctx context.Context, id string, input *AppInput) (*App, error) {
	resp, err := updateApp(ctx, client.Gql, id, *input)
	return appOrError(resp, err)
}

func (client *Client) DeleteApp(ctx context.Context, id string) error {
	resp, err := deleteApp(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) LookupApp(ctx context.Context, workspaceId, name string) (*App, error) {
	resp, err := lookupApp(ctx, client.Gql, workspaceId, name)
	if err != nil {
		return nil, err
	}
	switch len(resp.Apps) {
	case 0:
		return nil, errors.New("app not found")
	case 1:
		return &resp.Apps[0], nil
	default:
		return nil, errors.New("not implemented")
	}
}

func (client *Client) LookupModuleVersions(ctx context.Context, id string) ([]*ModuleVersion, error) {
	resp, err := lookupModuleVersions(ctx, client.Gql, id)
	if err != nil {
		return nil, err
	}
	// If the app exists, but there are no versions
	// published yet, return an error
	if len(resp.GetModuleVersions()) == 0 {
		return nil, errors.New("no module versions found")
	}
	return resp.GetModuleVersions(), nil
}

func (a *App) Oid() *oid.OID {
	return &oid.OID{
		Id:   a.Id,
		Type: oid.TypeApp,
	}
}
