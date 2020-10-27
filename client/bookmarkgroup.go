package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type BookmarkGroupPresentation = meta.BookmarkGroupPresentation

type BookmarkGroup struct {
	ID        string               `json:"id"`
	Workspace string               `json:"workspace"`
	Config    *BookmarkGroupConfig `json:"config"`
}

func (bg *BookmarkGroup) OID() *OID {
	return &OID{
		Type: TypeBookmarkGroup,
		ID:   bg.ID,
	}
}

type BookmarkGroupConfig struct {
	Name         string                     `json:"name"`
	Presentation *BookmarkGroupPresentation `json:"presentation"`
	IconURL      *string                    `json:"iconUrl"`
}

func (bg *BookmarkGroupConfig) toGQL() (*meta.BookmarkGroupInput, error) {
	bgInput := &meta.BookmarkGroupInput{
		Name:         &bg.Name,
		IconURL:      bg.IconURL,
		Presentation: bg.Presentation,
	}
	return bgInput, nil
}

func newBookmarkGroup(bg *meta.BookmarkGroup) (*BookmarkGroup, error) {
	bgconfig := &BookmarkGroupConfig{
		Name: bg.Name,
	}

	if bg.IconURL != "" {
		bgconfig.IconURL = &bg.IconURL
	}

	return &BookmarkGroup{
		ID:        bg.ID.String(),
		Workspace: bg.WorkspaceID.String(),
		Config:    bgconfig,
	}, nil
}
