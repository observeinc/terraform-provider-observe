package meta

import (
	"context"
	"errors"
)

var (
	backendAppFragment = `
	fragment appFields on App {
	    id
	    name
	    iconUrl
		description
		workspaceId
		folderId
		config {
			moduleId
			version
		}
		status {
			state
		}
	}`
)

func (c *Client) GetApp(ctx context.Context, id string) (*App, error) {
	result, err := c.Run(ctx, backendAppFragment+`
	query app($id: ObjectId!) {
		app(id: $id) {
			...appFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var app App
	if err = decodeStrict(getNested(result, "app"), &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) CreateApp(ctx context.Context, workspaceID string, config *AppInput) (*App, error) {
	result, err := c.Run(ctx, backendAppFragment+`
	mutation createApp($workspaceId: ObjectId!, $config: AppInput!) {
		createApp(workspaceId:$workspaceId, app: $config) {
			...appFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"config":      config,
	})
	if err != nil {
		return nil, err
	}

	var app App
	if err = decodeStrict(getNested(result, "createApp"), &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) UpdateApp(ctx context.Context, id string, config *AppInput) (*App, error) {
	result, err := c.Run(ctx, backendAppFragment+`
	mutation updateApp($id: ObjectId!, $config: AppInput!) {
		updateApp(id:$id, app: $config) {
			...appFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"config": config,
	})
	if err != nil {
		return nil, err
	}

	var app App
	if err = decodeStrict(getNested(result, "updateApp"), &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) DeleteApp(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteApp(id: $id) {
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
	nested := getNested(result, "deleteApp")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}

// LookupApp retrieves app by name.
// TODO: this should be bound to a folderId, not a workspace.
func (c *Client) LookupApp(ctx context.Context, workspaceId, name string) (*App, error) {
	result, err := c.Run(ctx, backendAppFragment+`
	query lookupApp($workspaceId: ObjectId!, $name: String!) {
		apps(workspaceId: $workspaceId, name: $name) {
			...appFields
		}
    }`, map[string]interface{}{
		"workspaceId": workspaceId,
		"name":        name,
	})

	if err != nil {
		return nil, err
	}

	var apps []*App
	if err := decodeStrict(getNested(result, "workspace", "apps"), &apps); err != nil {
		return nil, err
	}
	switch len(apps) {
	case 0:
		return nil, errors.New("app not found")
	case 1:
		return apps[0], nil
	default:
		return nil, errors.New("not implemented")
	}
}
