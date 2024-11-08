package meta

import (
	"context"
	"errors"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorActionResponse interface {
	GetMonitorAction() *MonitorAction
}

func monitorActionOrError(m monitorActionResponse, err error) (*MonitorAction, error) {
	if err != nil {
		return nil, err
	}
	return m.GetMonitorAction(), nil
}

func (client *Client) CreateMonitorAction(ctx context.Context, input *MonitorActionInput) (*MonitorAction, error) {
	resp, err := createMonitorAction(ctx, client.Gql, *input)
	return monitorActionOrError(resp, err)
}

func (client *Client) GetMonitorAction(ctx context.Context, id string) (*MonitorAction, error) {
	resp, err := getMonitorAction(ctx, client.Gql, id)
	return monitorActionOrError(resp, err)
}

func (client *Client) LookupMonitorAction(ctx context.Context, workspaceId, name string) (*MonitorAction, error) {
	resp, err := searchMonitorActions(ctx, client.Gql, &workspaceId, &name)
	if err != nil {
		return nil, err
	}
	switch len(resp.MonitorActions) {
	case 0:
		return nil, errors.New("monitor action not found")
	case 1:
		return &resp.MonitorActions[0], nil
	default:
		return nil, errors.New("not implemented")
	}
}

func (client *Client) UpdateMonitorAction(ctx context.Context, id string, input *MonitorActionInput) (*MonitorAction, error) {
	resp, err := updateMonitorAction(ctx, client.Gql, id, *input)
	return monitorActionOrError(resp, err)
}

func (client *Client) DeleteMonitorAction(ctx context.Context, id string) error {
	resp, err := deleteMonitorAction(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func MonitorActionOid(c MonitorAction) *oid.OID {
	return &oid.OID{
		Id:   c.GetId(),
		Type: oid.TypeMonitorAction,
	}
}
