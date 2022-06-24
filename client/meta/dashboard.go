package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type dashboard interface {
	GetDashboard() Dashboard
}

func dashboardOrError(d dashboard, err error) (*Dashboard, error) {
	if err != nil {
		return nil, err
	}
	result := d.GetDashboard()
	return &result, nil
}

func (client *Client) SaveDashboard(ctx context.Context, input *DashboardInput) (*Dashboard, error) {
	resp, err := saveDashboard(ctx, client.Gql, *input)
	return dashboardOrError(resp, err)
}

func (client *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	resp, err := getDashboard(ctx, client.Gql, id)
	return dashboardOrError(resp, err)
}

func (client *Client) DeleteDashboard(ctx context.Context, id string) error {
	resp, err := deleteDashboard(ctx, client.Gql, id)
	if err != nil {
		return err
	}
	return resultStatusError(resp, err)
}

func (d *Dashboard) Oid() *oid.OID {
	return &oid.OID{
		Id:   d.Id,
		Type: oid.TypeDashboard,
	}
}
