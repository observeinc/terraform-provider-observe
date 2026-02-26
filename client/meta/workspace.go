package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type workspaceResponse interface {
	GetWorkspace() *Workspace
}

func workspaceOrError(w workspaceResponse, err error) (*Workspace, error) {
	if err != nil {
		return nil, err
	}
	return w.GetWorkspace(), nil
}

func (client *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	resp, err := getWorkspace(ctx, client.Gql, id)
	return workspaceOrError(resp, err)
}

func (client *Client) LookupWorkspace(ctx context.Context, name string) (*Workspace, error) {
	resp, err := lookupWorkspace(ctx, client.Gql, name)
	return workspaceOrError(resp, err)
}

func (client *Client) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	resp, err := listWorkspaces(ctx, client.Gql)
	if err != nil {
		return nil, err
	}
	res := make([]*Workspace, 0)
	for _, workspace := range resp.Workspaces {
		res = append(res, &workspace)
	}
	return res, nil
}

func (w *Workspace) Oid() *oid.OID {
	return &oid.OID{
		Id:   w.Id,
		Type: oid.TypeWorkspace,
	}
}
