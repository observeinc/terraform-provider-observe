package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/api"
)

// Workspace acts as top-level grouping
type Workspace struct {
	ID       string            `json:"id"`
	Config   *WorkspaceConfig  `json:"config"`
	Datasets map[string]string `json:"datasets"`
}

func (w *Workspace) OID() *OID {
	return &OID{
		Type: TypeWorkspace,
		ID:   w.ID,
	}
}

// WorkspaceConfig contains configurable elements associated to Workspace
type WorkspaceConfig struct {
	Name string `json:"name"`
}

func newWorkspace(w *api.Workspace) (*Workspace, error) {
	ws := &Workspace{
		ID: w.ID.String(),
		Config: &WorkspaceConfig{
			Name: w.Label,
		},
		Datasets: make(map[string]string, len(w.Datasets)),
	}

	for _, gqlDataset := range w.Datasets {
		ws.Datasets[gqlDataset.Label] = gqlDataset.ID.String()
	}
	return ws, nil
}
