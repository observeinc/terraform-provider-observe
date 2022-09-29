package meta

import "context"

type dashboardLink interface {
	GetDashboardLink() DashboardLink
}

func dashboardLinkOrError(d dashboardLink, err error) (*DashboardLink, error) {
	if err != nil {
		return nil, err
	}
	result := d.GetDashboardLink()
	return &result, nil
}

func (client *Client) GetDashboardLink(ctx context.Context, id string) (*DashboardLink, error) {
	return dashboardLinkOrError(getDashboardLink(ctx, client.Gql, id))
}

func (client *Client) CreateDashboardLink(ctx context.Context, input DashboardLinkInput) (*DashboardLink, error) {
	return dashboardLinkOrError(createDashboardLink(ctx, client.Gql, input))
}

func (client *Client) UpdateDashboardLink(ctx context.Context, id string, input DashboardLinkInput) (*DashboardLink, error) {
	return dashboardLinkOrError(updateDashboardLink(ctx, client.Gql, id, input))
}

func (client *Client) DeleteDashboardLink(ctx context.Context, id string) error {
	return resultStatusError(deleteDashboardLink(ctx, client.Gql, id))
}
