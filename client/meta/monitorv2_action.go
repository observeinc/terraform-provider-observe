package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorV2ActionResponse interface {
	GetMonitorV2Action() MonitorV2Action
}

func monitorV2ActionOrError(m monitorV2ActionResponse, err error) (*MonitorV2Action, error) {
	if err != nil {
		return nil, err
	}
	result := m.GetMonitorV2Action()
	return &result, nil
}

func (client *Client) CreateMonitorV2Action(ctx context.Context, workspaceId string, input *MonitorV2ActionInput) (*MonitorV2Action, error) {
	resp, err := createMonitorV2Action(ctx, client.Gql, workspaceId, *input)
	return monitorV2ActionOrError(resp, err)
}

func (client *Client) GetMonitorV2Action(ctx context.Context, id string) (*MonitorV2Action, error) {
	resp, err := getMonitorV2Action(ctx, client.Gql, id)
	return monitorV2ActionOrError(resp, err)
}

func (client *Client) UpdateMonitorV2Action(ctx context.Context, id string, input *MonitorV2ActionInput) (*MonitorV2Action, error) {
	resp, err := updateMonitorV2Action(ctx, client.Gql, id, *input)
	return monitorV2ActionOrError(resp, err)
}

func (client *Client) DeleteMonitorV2Action(ctx context.Context, id string) error {
	resp, err := deleteMonitorV2Action(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) SearchMonitorV2Action(ctx context.Context, workspaceId *string, nameExact *string) (*MonitorV2Action, error) {
	resp, err := searchMonitorV2Action(ctx, client.Gql, workspaceId, nil, nameExact, nil)
	if err != nil || resp == nil || len(resp.MonitorV2Actions.Results) != 1 {
		return nil, err
	}
	return &resp.MonitorV2Actions.Results[0], nil
}

func (m *MonitorV2Action) Oid() *oid.OID {
	return &oid.OID{
		Id:   m.Id,
		Type: oid.TypeMonitorV2Action,
	}
}
