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
	if err != nil {
		return nil, err
	}
	ws := resp.GetWorksheet()
	return &ws, err
}

func (client *Client) GetWorksheet(ctx context.Context, id string) (*Worksheet, error) {
	resp, err := getWorksheet(ctx, client.Gql, id)
	return worksheetOrError(resp, err)
}

func (client *Client) ListWorksheetIdLabelOnly(ctx context.Context, workspaceId string) ([]*WorksheetIdLabel, error) {
	resp, err := listWorksheetsIdLabelOnly(ctx, client.Gql, workspaceId)
	if err != nil {
		return nil, err
	}
	result := make([]*WorksheetIdLabel, 0)
	for _, wks := range resp.WorksheetSearch.Worksheets {
		sheet := wks.Worksheet
		result = append(result, &sheet)
	}
	return result, nil
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
