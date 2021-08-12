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

func (c *Client) CreateWorkspace(ctx context.Context, config *WorkspaceInput) (*Workspace, error) {
	result, err := c.Run(ctx, backendWorkspaceFragment+`
	mutation createWorkspace($config: WorkspaceInput!) {
		createWorkspace(definition: $config) {
			...workspaceFields
		}
	}`, map[string]interface{}{
		"config": config,
	})
	if err != nil {
		return nil, err
	}

	var w Workspace
	if err = decodeStrict(getNested(result, "createWorkspace"), &w); err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, id string, config *WorkspaceInput) (*Workspace, error) {
	result, err := c.Run(ctx, backendWorkspaceFragment+`
	mutation updateWorkspace($id: ObjectId!, $config: WorkspaceInput!) {
		updateWorkspace(id:$id, definition: $config) {
			...workspaceFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"config": config,
	})
	if err != nil {
		return nil, err
	}

	var w Workspace
	if err = decodeStrict(getNested(result, "updateWorkspace"), &w); err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *Client) DeleteWorkspace(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteWorkspace(id: $id) {
            success
            errorMessage
            detailedInfo
        }
    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "deleteWorkspace")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
