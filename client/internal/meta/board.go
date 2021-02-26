package meta

import (
	"context"
)

var (
	backendBoardFragment = `
	fragment boardFields on Board {
	    id
		datasetId
	    name
		type
		board
	}`
)

func (c *Client) GetBoard(ctx context.Context, id string) (*Board, error) {
	result, err := c.Run(ctx, backendBoardFragment+`
	query getBoard($id: ObjectId!) {
		getBoard(id: $id) {
			...boardFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var b Board
	if err = decodeStrict(getNested(result, "getBoard"), &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (c *Client) CreateBoard(ctx context.Context, datasetId string, boardType BoardType, config *BoardInput) (*Board, error) {
	result, err := c.Run(ctx, backendBoardFragment+`
	mutation createBoard($datasetId: ObjectId!, $type: BoardType!, $board: BoardInput!) {
		createBoard(datasetId:$datasetId, type: $type, board: $board) {
			...boardFields
		}
	}`, map[string]interface{}{
		"datasetId": datasetId,
		"type":      boardType,
		"board":     config,
	})
	if err != nil {
		return nil, err
	}

	var b Board
	if err = decodeStrict(getNested(result, "createBoard"), &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (c *Client) UpdateBoard(ctx context.Context, id string, config *BoardInput) (*Board, error) {
	result, err := c.Run(ctx, backendBoardFragment+`
	mutation updateBoard($id: ObjectId!, $board: BoardInput!) {
		updateBoard(id:$id, board: $board) {
			...boardFields
		}
	}`, map[string]interface{}{
		"id":    id,
		"board": config,
	})
	if err != nil {
		return nil, err
	}

	var b Board
	if err = decodeStrict(getNested(result, "updateBoard"), &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (c *Client) DeleteBoard(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteBoard(id: $id) {
            success
            errorMessage
            detailedInfo
        }
    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "deleteBoard")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
