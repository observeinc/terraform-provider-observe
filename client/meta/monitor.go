package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorResponse interface {
	GetMonitor() Monitor
}

func monitorOrError(m monitorResponse, err error) (*Monitor, error) {
	if err != nil {
		return nil, err
	}
	result := m.GetMonitor()
	return &result, nil
}

func (client *Client) CreateMonitor(ctx context.Context, workspaceId string, input *MonitorInput) (*Monitor, error) {
	resp, err := createMonitor(ctx, client.Gql, workspaceId, *input)
	return monitorOrError(resp.Monitor, err)
}

func (client *Client) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	resp, err := getMonitor(ctx, client.Gql, id)
	return monitorOrError(resp, err)
}

func (client *Client) UpdateMonitor(ctx context.Context, id string, input *MonitorInput) (*Monitor, error) {
	resp, err := updateMonitor(ctx, client.Gql, id, *input)
	return monitorOrError(resp.Monitor, err)
}

func (client *Client) DeleteMonitor(ctx context.Context, id string) error {
	resp, err := deleteMonitor(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) LookupMonitor(ctx context.Context, workspaceId, name string) (*Monitor, error) {
	resp, err := lookupMonitor(ctx, client.Gql, workspaceId, name)
	return monitorOrError(resp.Monitor, err)
}

func (m *Monitor) Oid() *oid.OID {
	return &oid.OID{
		Id:   m.Id,
		Type: oid.TypeMonitor,
	}
}
