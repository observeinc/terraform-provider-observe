package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type boardResponse interface {
	GetBoard() Board
}

func boardOrError(b boardResponse, err error) (*Board, error) {
	if err != nil {
		return nil, err
	}
	result := b.GetBoard()
	return &result, nil
}

func (client *Client) GetBoard(ctx context.Context, id string) (*Board, error) {
	resp, err := getBoard(ctx, client.Gql, id)
	return boardOrError(resp, err)
}

func (client *Client) CreateBoard(ctx context.Context, datasetId string, boardType BoardType, config *BoardInput) (*Board, error) {
	resp, err := createBoard(ctx, client.Gql, datasetId, boardType, *config)
	return boardOrError(resp, err)
}

func (client *Client) UpdateBoard(ctx context.Context, id string, config *BoardInput) (*Board, error) {
	resp, err := updateBoard(ctx, client.Gql, id, *config)
	return boardOrError(resp, err)
}

func (client *Client) DeleteBoard(ctx context.Context, id string) error {
	resp, err := deleteBoard(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (b *Board) Oid() *oid.OID {
	return &oid.OID{
		Id:   b.Id,
		Type: oid.TypeBoard,
	}
}
