package meta

import (
	"context"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type datasetOutboundShareResponse interface {
	GetDatasetOutboundShare() DatasetOutboundShare
}

func datasetOutboundShareOrError(r datasetOutboundShareResponse, err error) (*DatasetOutboundShare, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetDatasetOutboundShare()
	return &result, nil
}

func (client *Client) GetDatasetOutboundShare(ctx context.Context, id string) (*DatasetOutboundShare, error) {
	resp, err := getDatasetOutboundShare(ctx, client.Gql, id)
	return datasetOutboundShareOrError(resp, err)
}

func (client *Client) CreateDatasetOutboundShare(ctx context.Context, workspaceId string, datasetId string, outboundShareId string, input *DatasetOutboundShareInput) (*DatasetOutboundShare, error) {
	resp, err := createDatasetOutboundShare(ctx, client.Gql, workspaceId, datasetId, outboundShareId, *input)
	return datasetOutboundShareOrError(resp, err)
}

func (client *Client) UpdateDatasetOutboundShare(ctx context.Context, id string, input *DatasetOutboundShareInput) (*DatasetOutboundShare, error) {
	resp, err := updateDatasetOutboundShare(ctx, client.Gql, id, *input)
	return datasetOutboundShareOrError(resp, err)
}

func (client *Client) DeleteDatasetOutboundShare(ctx context.Context, id string) error {
	resp, err := deleteDatasetOutboundShare(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (p *DatasetOutboundShare) Oid() *oid.OID {
	return &oid.OID{
		Id:   p.Id,
		Type: oid.TypeDatasetOutboundShare,
	}
}
