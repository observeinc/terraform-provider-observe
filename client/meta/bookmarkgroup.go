package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type bookmarkGroupResponse interface {
	GetBookmarkGroup() BookmarkGroup
}

func bookmarkGroupOrError(b bookmarkGroupResponse, err error) (*BookmarkGroup, error) {
	if err != nil {
		return nil, err
	}
	result := b.GetBookmarkGroup()
	return &result, nil
}

func (client *Client) CreateOrUpdateBookmarkGroup(ctx context.Context, id *string, input *BookmarkGroupInput) (*BookmarkGroup, error) {
	resp, err := createOrUpdateBookmarkGroup(ctx, client.Gql, id, *input)
	return bookmarkGroupOrError(resp, err)
}

func (client *Client) GetBookmarkGroup(ctx context.Context, id string) (*BookmarkGroup, error) {
	resp, err := getBookmarkGroup(ctx, client.Gql, id)
	return bookmarkGroupOrError(resp, err)
}

func (client *Client) DeleteBookmarkGroup(ctx context.Context, id string) error {
	resp, err := deleteBookmarkGroup(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (b *BookmarkGroup) Oid() *oid.OID {
	return &oid.OID{
		Id:   b.Id,
		Type: oid.TypeBookmarkGroup,
	}
}
