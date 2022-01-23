package meta

import (
	"context"
)

var (
	backendFolderFragment = `
	fragment folderFields on Folder {
	    id
	    name
	    iconUrl
		description
		workspaceId
	}`
)

func (c *Client) GetFolder(ctx context.Context, id string) (*Folder, error) {
	result, err := c.Run(ctx, backendFolderFragment+`
	query folder($id: ObjectId!) {
		folder(id: $id) {
			...folderFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var f Folder
	if err = decodeStrict(getNested(result, "folder"), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

func (c *Client) CreateFolder(ctx context.Context, workspaceID string, config *FolderInput) (*Folder, error) {
	result, err := c.Run(ctx, backendFolderFragment+`
	mutation createFolder($workspaceId: ObjectId!, $config: FolderInput!) {
		createFolder(workspaceId:$workspaceId, folder: $config) {
			...folderFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"config":      config,
	})
	if err != nil {
		return nil, err
	}

	var f Folder
	if err = decodeStrict(getNested(result, "createFolder"), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

func (c *Client) UpdateFolder(ctx context.Context, id string, config *FolderInput) (*Folder, error) {
	result, err := c.Run(ctx, backendFolderFragment+`
	mutation updateFolder($id: ObjectId!, $config: FolderInput!) {
		updateFolder(id:$id, folder: $config) {
			...folderFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"config": config,
	})
	if err != nil {
		return nil, err
	}

	var f Folder
	if err = decodeStrict(getNested(result, "updateFolder"), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

func (c *Client) DeleteFolder(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteFolder(id: $id) {
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
	nested := getNested(result, "deleteFolder")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}

// LookupFolder retrieves folder by name.
func (c *Client) LookupFolder(ctx context.Context, workspaceId, name string) (*Folder, error) {
	result, err := c.Run(ctx, backendFolderFragment+`
	query lookupFolder($workspaceId: ObjectId!, $name: String!) {
		workspace(id: $workspaceId) {
			folder(name: $name) {
            	...folderFields
        	}
		}
    }`, map[string]interface{}{
		"workspaceId": workspaceId,
		"name":        name,
	})

	if err != nil {
		return nil, err
	}

	var folder Folder
	if err := decodeStrict(getNested(result, "workspace", "folder"), &folder); err != nil {
		return nil, err
	}
	return &folder, nil
}
