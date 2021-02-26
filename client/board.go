package client

import (
	"encoding/json"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type (
	BoardType = meta.BoardType
)

var (
	BoardTypes = meta.AllBoardType
)

type Board struct {
	ID      string       `json:"id"`
	Dataset *OID         `json:"dataset"`
	Type    BoardType    `json:"type"`
	Config  *BoardConfig `json:"config"`
}

func (b *Board) OID() *OID {
	return &OID{
		Type: TypeBoard,
		ID:   b.ID,
	}
}

type BoardConfig struct {
	Name string `json:"name"`
	JSON string `json:"json"`
}

func (bc *BoardConfig) toGQL() (*meta.BoardInput, error) {
	b := &meta.BoardInput{
		Name:  &bc.Name,
		Board: &bc.JSON,
	}
	return b, nil
}

func newBoard(b *meta.Board) (*Board, error) {
	data, err := json.Marshal(b.Board)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board: %w", err)
	}

	bc := &BoardConfig{
		Name: b.Name,
		JSON: string(data),
	}

	return &Board{
		ID: b.ID.String(),
		Dataset: &OID{
			Type: TypeDataset,
			ID:   b.DatasetID.String(),
		},
		Type:   b.Type,
		Config: bc,
	}, nil
}
