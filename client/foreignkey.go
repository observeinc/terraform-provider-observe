package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/api"
)

type ForeignKey struct {
	ID        string            `json:"id"`
	Workspace string            `json:"workspace"`
	Config    *ForeignKeyConfig `json:"config"`
}

type ForeignKeyConfig struct {
	Source    *string  `json:"source"`
	Target    *string  `json:"target"`
	SrcFields []string `json:"srcFields"`
	DstFields []string `json:"srcFields"`
	Label     *string  `json:"label"`
}

func (fk *ForeignKeyConfig) toGQL() (*api.DeferredForeignKeyInput, error) {
	dfkInput := &api.DeferredForeignKeyInput{
		SourceDataset: api.DeferredDatasetReferenceInput{
			DatasetID: toObjectPointer(fk.Source),
		},
		TargetDataset: api.DeferredDatasetReferenceInput{
			DatasetID: toObjectPointer(fk.Target),
		},
		SrcFields: fk.SrcFields,
		DstFields: fk.DstFields,
		Label:     fk.Label,
	}

	return dfkInput, nil
}

func newForeignKey(dfk *api.DeferredForeignKey) (*ForeignKey, error) {
	fkconfig := &ForeignKeyConfig{
		SrcFields: dfk.SrcFields,
		DstFields: dfk.DstFields,
		Label:     dfk.Label,
	}

	if dfk.SourceDataset.DatasetID != nil {
		s := dfk.SourceDataset.DatasetID.String()
		fkconfig.Source = &s
	}

	if dfk.TargetDataset.DatasetID != nil {
		s := dfk.TargetDataset.DatasetID.String()
		fkconfig.Target = &s
	}

	return &ForeignKey{
		ID:        dfk.ID.String(),
		Workspace: dfk.WorkspaceID.String(),
		Config:    fkconfig,
	}, nil
}
