package meta

import (
	"context"
)

var (
	backendWorkspaceFragment = `
	fragment workspaceFields on Project {
		id
		label
		datasets {
			id
			label
		}
	}`
)

func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	result, err := c.Run(ctx, backendWorkspaceFragment+`
	query getWorkspace($id: ObjectId!) {
		workspace(id: $id) {
			...workspaceFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err = decodeStrict(getNested(result, "workspace"), &workspace); err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (c *Client) LookupWorkspace(ctx context.Context, name string) (*Workspace, error) {
	result, err := c.Run(ctx, backendWorkspaceFragment+`
	query lookupWorkspace($name: String!) {
		workspace(label: $name) {
			...workspaceFields
		}
	}`, map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err = decodeStrict(getNested(result, "workspace"), &workspace); err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	result, err := c.Run(ctx, backendWorkspaceFragment+`
	query ListWorkspaces() {
		projects {
			...workspaceFields
		}
	}`, nil)
	if err != nil {
		return nil, err
	}

	var workspaces []*Workspace
	if err = decodeStrict(getNested(result, "projects"), &workspaces); err != nil {
		return nil, err
	}

	return workspaces, nil
}
