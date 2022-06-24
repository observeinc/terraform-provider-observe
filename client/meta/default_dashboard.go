package meta

import (
	"context"
)

func (client *Client) SetDefaultDashboard(ctx context.Context, dsid string, dashid string) error {
	resp, err := setDefaultDashboard(ctx, client.Gql, dsid, dashid)
	return resultStatusError(resp, err)
}

func (client *Client) GetDefaultDashboard(ctx context.Context, id string) (*string, error) {
	resp, err := getDefaultDashboard(ctx, client.Gql, id)
	return resp.DefaultDashboard, err
}

func (client *Client) ClearDefaultDashboard(ctx context.Context, id string) error {
	resp, err := clearDefaultDashboard(ctx, client.Gql, id)
	if err != nil {
		return err
	}
	return resultStatusError(resp, err)
}
