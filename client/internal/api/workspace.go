package api

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

func (c *Client) GetWorkspace(id string) (*Workspace, error) {
	result, err := c.Run(backendWorkspaceFragment+`
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

func (c *Client) ListWorkspaces() ([]*Workspace, error) {
	result, err := c.Run(backendWorkspaceFragment+`
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
