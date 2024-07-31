package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorV2DestinationResponse interface {
	GetMonitorV2Destination() MonitorV2Destination
}

func monitorV2DestinationOrError(m monitorV2DestinationResponse, err error) (*MonitorV2Destination, error) {
	if err != nil {
		return nil, err
	}
	result := m.GetMonitorV2Destination()
	return &result, nil
}

func (client *Client) CreateMonitorV2Destination(ctx context.Context, workspaceId string, input *MonitorV2DestinationInput) (*MonitorV2Destination, error) {
	resp, err := createMonitorV2Destination(ctx, client.Gql, workspaceId, *input)
	return monitorV2DestinationOrError(resp, err)
}

func (client *Client) GetMonitorV2Destination(ctx context.Context, id string) (*MonitorV2Destination, error) {
	resp, err := getMonitorV2Destination(ctx, client.Gql, id)
	return monitorV2DestinationOrError(resp, err)
}

func (client *Client) UpdateMonitorV2Destination(ctx context.Context, id string, input *MonitorV2DestinationInput) (*MonitorV2Destination, error) {
	resp, err := updateMonitorV2Destination(ctx, client.Gql, id, *input)
	return monitorV2DestinationOrError(resp, err)
}

func (client *Client) DeleteMonitorV2Destination(ctx context.Context, id string) error {
	resp, err := deleteMonitorV2Destination(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (m *MonitorV2Destination) Oid() *oid.OID {
	return &oid.OID{
		Id:   m.Id,
		Type: oid.TypeMonitorV2Destination,
	}
}
