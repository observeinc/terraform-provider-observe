package client

import (
	"encoding/json"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Worksheet struct {
	ID        string           `json:"id"`
	Workspace *OID             `json:"workspace"`
	Config    *WorksheetConfig `json:"config"`
}

func (b *Worksheet) OID() *OID {
	return &OID{
		Type: TypeWorksheet,
		ID:   b.ID,
	}
}

type WorksheetConfig struct {
	Name    string  `json:"name"`
	IconURL *string `json:"icon_url"`
	Queries *string `json:"queries"`
}

func (wc *WorksheetConfig) toGQL() (*meta.WorksheetInput, error) {
	w := &meta.WorksheetInput{
		Label: wc.Name,
		Icon:  wc.IconURL,
	}

	if wc.Queries != nil {
		var v []interface{}
		if err := json.Unmarshal([]byte(*wc.Queries), &v); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		decoder, err := meta.NewDecoder(true, &w.Queries)
		if err != nil {
			return nil, fmt.Errorf("failed to create decoder: %w", err)
		}

		if err := decoder.Decode(v); err != nil {
			return nil, fmt.Errorf("failed to map JSON to worksheet: %w", err)
		}
	}

	return w, nil
}

func newWorksheet(w *meta.Worksheet) (*Worksheet, error) {
	wc := &WorksheetConfig{
		Name:    w.Label,
		IconURL: w.Icon,
	}

	{
		data, err := json.Marshal(w.Queries)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal queries: %w", err)
		}
		s := string(data)
		wc.Queries = &s
	}

	return &Worksheet{
		ID: w.ID.String(),
		Workspace: &OID{
			Type: TypeWorkspace,
			ID:   w.Workspace.ID.String(),
		},
		Config: wc,
	}, nil
}
