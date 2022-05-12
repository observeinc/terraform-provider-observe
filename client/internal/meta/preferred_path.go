package meta

import (
	"context"
)

var (
	backendPreferredPathFragment = `
	fragment preferredPathFields on PreferredPath {
		id
		name
		iconUrl
		description
		workspaceId
		folderId
		sourceDataset
		path {
			linkId
			reverse
		}
	}

	fragment preferredPathWithStatusFields on PreferredPathWithStatus {
		path {
			...preferredPathFields
		}
		error
	}
	`
)

func (c *Client) GetPreferredPath(ctx context.Context, id string) (*PreferredPathWithStatus, error) {
	result, err := c.Run(ctx, backendPreferredPathFragment+`
	query preferredPath($id: ObjectId!) {
		preferredPath(id: $id) {
			...preferredPathWithStatusFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var p PreferredPathWithStatus
	if err = decodeStrict(getNested(result, "preferredPath"), &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (c *Client) CreatePreferredPath(ctx context.Context, workspaceID string, config *PreferredPathInput) (*PreferredPathWithStatus, error) {
	result, err := c.Run(ctx, backendPreferredPathFragment+`
	mutation createPreferredPath($workspaceId: ObjectId!, $config: PreferredPathInput!) {
		createPreferredPath(workspaceId:$workspaceId, path: $config) {
			...preferredPathWithStatusFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"config":      config,
	})
	if err != nil {
		return nil, err
	}

	var p PreferredPathWithStatus
	if err = decodeStrict(getNested(result, "createPreferredPath"), &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (c *Client) UpdatePreferredPath(ctx context.Context, id string, config *PreferredPathInput) (*PreferredPathWithStatus, error) {
	result, err := c.Run(ctx, backendPreferredPathFragment+`
	mutation updatePreferredPath($id: ObjectId!, $config: PreferredPathInput!) {
		updatePreferredPath(id:$id, path: $config) {
			...preferredPathWithStatusFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"config": config,
	})
	if err != nil {
		return nil, err
	}

	var p PreferredPathWithStatus
	if err = decodeStrict(getNested(result, "updatePreferredPath"), &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (c *Client) DeletePreferredPath(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deletePreferredPath(id: $id) {
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
	nested := getNested(result, "deletePreferredPath")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
