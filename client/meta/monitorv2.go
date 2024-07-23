package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorV2Response interface {
	GetMonitorV2() MonitorV2
}

func monitorV2OrError(m monitorV2Response, err error) (*MonitorV2, error) {
	if err != nil {
		return nil, err
	}
	result := m.GetMonitorV2()
	return &result, nil
}

func (client *Client) CreateMonitorV2(ctx context.Context, workspaceId string, input *MonitorV2Input) (*MonitorV2, error) {
	resp, err := createMonitorV2(ctx, client.Gql, workspaceId, *input)
	return monitorV2OrError(resp, err)
}

func (client *Client) GetMonitorV2(ctx context.Context, id string) (*MonitorV2, error) {
	resp, err := getMonitorV2(ctx, client.Gql, id)
	return monitorV2OrError(resp, err)
}

func (client *Client) UpdateMonitorV2(ctx context.Context, id string, input *MonitorV2Input) (*MonitorV2, error) {
	resp, err := updateMonitorV2(ctx, client.Gql, id, *input)
	return monitorV2OrError(resp, err)
}

func (client *Client) DeleteMonitorV2(ctx context.Context, id string) error {
	resp, err := deleteMonitorV2(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) LookupMonitorV2(ctx context.Context, workspaceId *string, folderId *string, nameExact *string, nameSubstring *string) (*MonitorV2, error) {
	resp, err := lookupMonitorV2(ctx, client.Gql, workspaceId, folderId, nameExact, nameSubstring)
	if err != nil || resp == nil || len(resp.MonitorV2s.Results) != 1 {
		return nil, err
	}
	return &resp.MonitorV2s.Results[0], nil
}

func (m *MonitorV2) Oid() *oid.OID {
	return &oid.OID{
		Id:   m.Id,
		Type: oid.TypeMonitorV2,
	}
}
