package client

import (
	"strings"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Bookmark struct {
	ID     string          `json:"id"`
	Config *BookmarkConfig `json:"config"`

	// backend returns targetIdKind. Use it to reconstruct target OID
	kind string
}

func (bm *Bookmark) OID() *OID {
	return &OID{
		Type: TypeBookmark,
		ID:   bm.ID,
	}
}

func (bm *Bookmark) GroupOID() *OID {
	return &OID{
		Type: TypeBookmarkGroup,
		ID:   bm.Config.GroupID,
	}
}

func (bm *Bookmark) TargetOID() *OID {
	// TODO: overly permissive, we're assuming backend always returns a valid
	// type here. Revisit when we consolidate Type with ObjectKind
	return &OID{
		Type: Type(strings.ToLower(bm.kind)),
		ID:   bm.Config.TargetID,
	}
}

type BookmarkConfig struct {
	Name     string  `json:"name"`
	IconURL  *string `json:"iconUrl"`
	TargetID string  `json:"targetId"`
	GroupID  string  `json:"groupId"`
}

func (bm *BookmarkConfig) toGQL() (*meta.BookmarkInput, error) {
	bmInput := &meta.BookmarkInput{
		Name:     &bm.Name,
		IconURL:  bm.IconURL,
		TargetID: toObjectPointer(&bm.TargetID),
		GroupID:  toObjectPointer(&bm.GroupID),
	}
	return bmInput, nil
}

func newBookmark(bm *meta.Bookmark) (*Bookmark, error) {
	bmconfig := &BookmarkConfig{
		Name:     bm.Name,
		TargetID: bm.TargetID.String(),
		GroupID:  bm.GroupID.String(),
	}

	if bm.IconURL != "" {
		bmconfig.IconURL = &bm.IconURL
	}

	return &Bookmark{
		ID:     bm.ID.String(),
		kind:   bm.TargetIDKind,
		Config: bmconfig,
	}, nil
}
