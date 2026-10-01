package meta

import (
	"context"
)

func (client *Client) SetDefaultDashboardForTag(ctx context.Context, tag string, dashid string) error {
	resp, err := setDefaultDashboardForTag(ctx, client.Gql, tag, dashid)
	return resultStatusError(resp, err)
}

func (client *Client) GetDefaultDashboardForTag(ctx context.Context, tag string) (*string, error) {
	resp, err := getDefaultDashboardForTag(ctx, client.Gql, tag)
	if err != nil {
		return nil, err
	}
	return resp.DefaultDashboardForTag, nil
}

// ClearDefaultDashboardForTag unbinds the default dashboard for tag by calling
// setDefaultDashboardForTag with dashboardId 0, which deletes the model_tag row.
func (client *Client) ClearDefaultDashboardForTag(ctx context.Context, tag string) error {
	return client.SetDefaultDashboardForTag(ctx, tag, "0")
}
