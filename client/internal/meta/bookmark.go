package meta

import (
	"context"
)

var (
	backendBookmarkFragment = `
	fragment bookmarkFields on Bookmark {
	    id
	    name
	    iconUrl
	    targetId
	    targetIdKind
	    groupId
	}`
)

func (c *Client) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	result, err := c.Run(ctx, backendBookmarkFragment+`
	query getBookmark($id: ObjectId!) {
		bookmark(id: $id) {
			...bookmarkFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var bg Bookmark
	if err = decodeStrict(getNested(result, "bookmark"), &bg); err != nil {
		return nil, err
	}

	return &bg, nil
}

func (c *Client) CreateOrUpdateBookmark(ctx context.Context, id *string, config *BookmarkInput) (*Bookmark, error) {
	result, err := c.Run(ctx, backendBookmarkFragment+`
	mutation createOrUpdateBookmark($id: ObjectId, $bookmark: BookmarkInput!) {
		createOrUpdateBookmark(id:$id, bookmark: $bookmark) {
			...bookmarkFields
		}
	}`, map[string]interface{}{
		"id":       id,
		"bookmark": config,
	})
	if err != nil {
		return nil, err
	}

	var bg Bookmark
	if err = decodeStrict(getNested(result, "createOrUpdateBookmark"), &bg); err != nil {
		return nil, err
	}

	return &bg, nil
}

func (c *Client) DeleteBookmark(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteBookmark(id: $id) {
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
	nested := getNested(result, "deleteBookmark")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
