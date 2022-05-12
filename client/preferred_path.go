package client

import (
	"errors"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type PreferredPath struct {
	ID        string               `json:"id"`
	Workspace string               `json:"workspace"`
	Config    *PreferredPathConfig `json:"config"`
}

type PreferredPathConfig struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Source      *OID                `json:"source"`
	Folder      *OID                `json:"folder"`
	Path        []PreferredPathStep `json:"path"`
}

type PreferredPathStep struct {
	Link     *OID    `json:"link"`
	Reverse  bool    `json:"reverse"`
	LinkName *string `json:"linkName"`
}

func (pp *PreferredPathConfig) toGQL() (*meta.PreferredPathInput, error) {
	if pp.Source == nil {
		return nil, errors.New("missing source dataset")
	}

	if t := pp.Source.Type; t != TypeDataset {
		return nil, fmt.Errorf("source dataset has wrong type %s", t)
	}

	ppInput := &meta.PreferredPathInput{
		Name:          pp.Name,
		Description:   pp.Description,
		FolderID:      toObjectPointer(pp.Folder.Version),
		SourceDataset: toObjectPointer(&pp.Source.ID),
	}

	for i, step := range pp.Path {
		if step.Link == nil {
			return nil, fmt.Errorf("missing link in step %d", i)
		}

		if t := step.Link.Type; t != TypeLink {
			return nil, fmt.Errorf("link has wrong type %s", t)
		}

		ppInput.Path = append(ppInput.Path, meta.PreferredPathStepInput{
			LinkId:   toObjectPointer(&step.Link.ID),
			Reverse:  step.Reverse,
			LinkName: step.LinkName,
		})
	}

	return ppInput, nil
}

func newPreferredPath(ps *meta.PreferredPathWithStatus) (*PreferredPath, error) {
	if ps.Error != nil {
		return nil, errors.New(*ps.Error)
	}

	folderId := ps.Path.FolderID.String()
	ppConfig := &PreferredPathConfig{
		Name:        ps.Path.Name,
		Description: ps.Path.Description,
		Folder: &OID{
			Type:    TypeFolder,
			ID:      ps.Path.WorkspaceID.String(),
			Version: &folderId,
		},
	}
	return &PreferredPath{
		ID:        ps.Path.Id.String(),
		Workspace: ps.Path.WorkspaceID.String(),
		Config:    ppConfig,
	}, nil
}
