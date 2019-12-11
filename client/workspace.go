package client

import (
	"context"
	"errors"

	"github.com/machinebox/graphql"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

type Workspace struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type backendWorkspace = Workspace

func (w *backendWorkspace) Convert() (*Workspace, error) {
	if w == nil {
		return nil, ErrWorkspaceNotFound
	}

	return &Workspace{
		ID:    w.ID,
		Label: w.Label,
	}, nil
}

func (c *Client) ListWorkspaces() ([]*Workspace, error) {
	req := graphql.NewRequest(`
	query {
	  projects {
		id
		label
	  }
	}`)

	var respData struct {
		Projects []*backendWorkspace `json:"projects"`
	}

	if err := c.client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	var result []*Workspace
	for _, i := range respData.Projects {
		el, err := i.Convert()
		if err != nil {
			return result, err
		}
		result = append(result, el)
	}
	return result, nil
}

func (c *Client) GetWorkspace(id string) (*Workspace, error) {
	req := graphql.NewRequest(`
	query GetWorkspace($id: ObjectId!){
	  workspace(id:$id) {
		id
		label
	  }
	}`)

	req.Var("id", id)

	var respData struct {
		Workspace *backendWorkspace `json:"workspace"`
	}

	if err := c.client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.Workspace.Convert()
}
