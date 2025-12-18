package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

func (client *Client) CreateMonitorV2MuteRule(ctx context.Context, workspaceId string, input *MonitorV2MuteRuleInput) (*MonitorV2MuteRule, error) {
	resp, err := createMonitorV2MuteRule(ctx, client.Gql, workspaceId, *input)
	if err != nil {
		return nil, err
	}
	return &resp.MuteRule, nil
}

func (client *Client) GetMonitorV2MuteRule(ctx context.Context, id string) (*MonitorV2MuteRule, error) {
	resp, err := getMonitorV2MuteRule(ctx, client.Gql, id)
	if err != nil {
		return nil, err
	}
	return &resp.MuteRule, nil
}

func (client *Client) UpdateMonitorV2MuteRule(ctx context.Context, id string, input *MonitorV2MuteRuleInput) (*MonitorV2MuteRule, error) {
	resp, err := updateMonitorV2MuteRule(ctx, client.Gql, id, *input)
	if err != nil {
		return nil, err
	}
	return &resp.MuteRule, nil
}

func (client *Client) DeleteMonitorV2MuteRule(ctx context.Context, id string) error {
	resp, err := deleteMonitorV2MuteRule(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) SearchMonitorV2MuteRule(ctx context.Context, workspaceId *string, nameExact *string) ([]MonitorV2MuteRule, error) {
	resp, err := searchMonitorV2MuteRule(ctx, client.Gql, workspaceId, nil, nameExact, nil)
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.SearchResult.Results, nil
}

func (m *MonitorV2MuteRule) Oid() *oid.OID {
	return &oid.OID{
		Id:   m.Id,
		Type: oid.TypeMonitorV2MuteRule,
	}
}
