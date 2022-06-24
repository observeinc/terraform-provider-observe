package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type bookmarkResponse interface {
	GetBookmark() Bookmark
}

func bookmarkOrError(b bookmarkResponse, err error) (*Bookmark, error) {
	if err != nil {
		return nil, err
	}
	result := b.GetBookmark()
	return &result, nil
}

func (client *Client) CreateOrUpdateBookmark(ctx context.Context, id *string, input *BookmarkInput) (*Bookmark, error) {
	resp, err := createOrUpdateBookmark(ctx, client.Gql, id, *input)
	return bookmarkOrError(resp, err)
}

func (client *Client) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	resp, err := getBookmark(ctx, client.Gql, id)
	return bookmarkOrError(resp, err)
}

func (client *Client) DeleteBookmark(ctx context.Context, id string) error {
	resp, err := deleteBookmark(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (b *Bookmark) Oid() *oid.OID {
	return &oid.OID{
		Id:   b.Id,
		Type: oid.TypeBookmark,
	}
}
