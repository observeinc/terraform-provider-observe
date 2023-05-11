package meta

import (
	"context"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type monitorActionAttachmentResponse interface {
	GetMonitorActionAttachment() *MonitorActionAttachment
}

func monitorActionAttachmentOrError(m monitorActionAttachmentResponse, err error) (*MonitorActionAttachment, error) {
	if err != nil {
		return nil, err
	}
	return m.GetMonitorActionAttachment(), nil
}

func (client *Client) CreateMonitorActionAttachment(ctx context.Context, input *MonitorActionAttachmentInput) (*MonitorActionAttachment, error) {
	resp, err := createMonitorActionAttachment(ctx, client.Gql, *input)
	return monitorActionAttachmentOrError(resp, err)
}

func (client *Client) GetMonitorActionAttachment(ctx context.Context, id string) (*MonitorActionAttachment, error) {
	resp, err := getMonitorActionAttachment(ctx, client.Gql, id)
	return monitorActionAttachmentOrError(resp, err)
}

func (client *Client) UpdateMonitorActionAttachment(ctx context.Context, id string, input *MonitorActionAttachmentInput) (*MonitorActionAttachment, error) {
	resp, err := updateMonitorActionAttachment(ctx, client.Gql, id, *input)
	return monitorActionAttachmentOrError(resp, err)
}

func (client *Client) DeleteMonitorActionAttachment(ctx context.Context, id string) error {
	resp, err := deleteMonitorActionAttachment(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func MonitorActionAttachmentOid(c MonitorActionAttachment) *oid.OID {
	return &oid.OID{
		Id:   c.GetId(),
		Type: oid.TypeMonitorActionAttachment,
	}
}
