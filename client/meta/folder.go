package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type folderResponse interface {
	GetFolder() Folder
}

func folderOrError(f folderResponse, err error) (*Folder, error) {
	if err != nil {
		return nil, err
	}
	result := f.GetFolder()
	return &result, nil
}

func (client *Client) CreateFolder(ctx context.Context, workspaceId string, input *FolderInput) (*Folder, error) {
	resp, err := createFolder(ctx, client.Gql, workspaceId, *input)
	return folderOrError(resp, err)
}

func (client *Client) GetFolder(ctx context.Context, id string) (*Folder, error) {
	resp, err := getFolder(ctx, client.Gql, id)
	return folderOrError(resp, err)
}

func (client *Client) UpdateFolder(ctx context.Context, id string, input *FolderInput) (*Folder, error) {
	resp, err := updateFolder(ctx, client.Gql, id, *input)
	return folderOrError(resp, err)
}

func (client *Client) DeleteFolder(ctx context.Context, id string) error {
	resp, err := deleteFolder(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) LookupFolder(ctx context.Context, workspaceId, name string) (*Folder, error) {
	resp, err := lookupFolder(ctx, client.Gql, workspaceId, name)
	return folderOrError(resp.Folder, err)
}

func (f *Folder) Oid() *oid.OID {
	// Shameful hack: Use the workspace ID as the ID, and use the actual folder ID as the version
	return &oid.OID{
		Id:      f.WorkspaceId,
		Type:    oid.TypeFolder,
		Version: &f.Id,
	}
}
