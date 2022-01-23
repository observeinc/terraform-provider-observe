package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Folder struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace"`
	Config      *FolderConfig `json:"config"`
}

type FolderConfig struct {
	Name        string  `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`
}

func (f *Folder) OID() *OID {
	return &OID{
		Type: TypeFolder,
		ID:   f.ID,
	}
}

func (config *FolderConfig) toGQL() (*meta.FolderInput, error) {
	folderInput := &meta.FolderInput{
		Name:        config.Name,
		IconURL:     config.IconURL,
		Description: config.Description,
	}

	return folderInput, nil
}

func newFolder(c *meta.Folder) (*Folder, error) {
	config := &FolderConfig{
		Name:        c.Name,
		IconURL:     c.IconURL,
		Description: c.Description,
	}

	return &Folder{
		ID:          c.ID.String(),
		WorkspaceID: c.WorkspaceId.String(),
		Config:      config,
	}, nil
}
