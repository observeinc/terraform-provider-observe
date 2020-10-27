package meta

import (
	"context"
)

var (
	backendBookmarkGroupFragment = `
	fragment bookmarkGroupFields on BookmarkGroup {
	    id
	    name
	    iconUrl
	    workspaceId
	}`
)

func (c *Client) GetBookmarkGroup(ctx context.Context, id string) (*BookmarkGroup, error) {
	result, err := c.Run(ctx, backendBookmarkGroupFragment+`
	query getBookmarkGroup($id: ObjectId!) {
		bookmarkGroup(id: $id) {
			...bookmarkGroupFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var bg BookmarkGroup
	if err = decodeStrict(getNested(result, "bookmarkGroup"), &bg); err != nil {
		return nil, err
	}

	return &bg, nil
}

func (c *Client) CreateOrUpdateBookmarkGroup(ctx context.Context, id *string, config *BookmarkGroupInput) (*BookmarkGroup, error) {
	result, err := c.Run(ctx, backendBookmarkGroupFragment+`
	mutation createOrUpdateBookmarkGroup($id: ObjectId, $data: BookmarkGroupInput!) {
		createOrUpdateBookmarkGroup(id:$id, group: $data) {
			...bookmarkGroupFields
		}
	}`, map[string]interface{}{
		"id":   id,
		"data": config,
	})
	if err != nil {
		return nil, err
	}

	var bg BookmarkGroup
	if err = decodeStrict(getNested(result, "createOrUpdateBookmarkGroup"), &bg); err != nil {
		return nil, err
	}

	return &bg, nil
}

func (c *Client) DeleteBookmarkGroup(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteBookmarkGroup(id: $id) {
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
	nested := getNested(result, "deleteBookmarkGroup")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
