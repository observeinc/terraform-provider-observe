package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type worksheetResponse interface {
	GetWorksheet() *Worksheet
}

func worksheetOrError(w worksheetResponse, err error) (*Worksheet, error) {
	if err != nil {
		return nil, err
	}
	return w.GetWorksheet(), nil
}

func (client *Client) SaveWorksheet(ctx context.Context, input *WorksheetInput) (*Worksheet, error) {
	resp, err := saveWorksheet(ctx, client.Gql, *input)
	// the schema appears to have changed so that saveworksheet returns a required worksheet now instead of an optional one
	if err != nil {
		return nil, err
	}
	worksheet := resp.GetWorksheet()
	return &worksheet, nil
}

func (client *Client) GetWorksheet(ctx context.Context, id string) (*Worksheet, error) {
	resp, err := getWorksheet(ctx, client.Gql, id)
	return worksheetOrError(resp, err)
}

func (client *Client) DeleteWorksheet(ctx context.Context, id string) error {
	resp, err := deleteWorksheet(ctx, client.Gql, id)
	return optionalResultStatusError(resp, err)
}

func (w *Worksheet) Oid() *oid.OID {
	return &oid.OID{
		Id:   w.Id,
		Type: oid.TypeWorksheet,
	}
}
