package client

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNotFound = errors.New("not found")

	flagObs2110 = "obs2110" // when set, allow concurrent API calls for foreign keys
)

// GetDataset returns dataset by ID
func (c *Client) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	result, err := c.metaAPI.GetDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// CreateDataset creates dataset
func (c *Client) CreateDataset(ctx context.Context, workspaceId string, config *DatasetConfig) (*Dataset, error) {
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.SaveDataset(ctx, workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// UpdateDataset updates existing dataset
func (c *Client) UpdateDataset(ctx context.Context, workspaceId string, id string, config *DatasetConfig) (*Dataset, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	datasetInput.ID = toObjectPointer(&id)
	result, err := c.metaAPI.SaveDataset(ctx, workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// DeleteDataset by ID
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.metaAPI.DeleteDataset(ctx, id)
}

// GetWorkspace by ID
func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	result, err := c.metaAPI.GetWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return newWorkspace(result)
}

// LookupWorkspace by name.
func (c *Client) LookupWorkspace(ctx context.Context, name string) (*Workspace, error) {
	workspaces, err := c.metaAPI.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup workspace: %w", err)
	}

	for _, w := range workspaces {
		if w.Label == name {
			return newWorkspace(w)
		}
	}
	// TODO: return not found?
	return nil, ErrNotFound
}

// ListWorkspaces.
func (c *Client) ListWorkspaces(ctx context.Context) (workspaces []*Workspace, err error) {
	result, err := c.metaAPI.ListWorkspaces(ctx)
	if err != nil {
		return
	}

	for _, w := range result {
		if ws, err := newWorkspace(w); err != nil {
			return nil, err
		} else {
			workspaces = append(workspaces, ws)
		}
	}

	return
}

// LookupDataset by name.
func (c *Client) LookupDataset(ctx context.Context, workspaceID string, name string) (*Dataset, error) {
	workspace, err := c.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup dataset: %w", err)
	}

	id, ok := workspace.Datasets[name]
	if !ok {
		return nil, ErrNotFound
	}
	return c.GetDataset(ctx, id)
}

// CreateForeignKey
func (c *Client) CreateForeignKey(ctx context.Context, workspaceID string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.CreateDeferredForeignKey(ctx, workspaceID, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		// call internal API directly since DeleteForeignKey() acquires lock
		c.metaAPI.DeleteDeferredForeignKey(ctx, result.ID.String())
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// UpdateForeignKey by ID
func (c *Client) UpdateForeignKey(ctx context.Context, id string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.UpdateDeferredForeignKey(ctx, id, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// GetForeignKey returns deferred foreign key
func (c *Client) GetForeignKey(ctx context.Context, id string) (*ForeignKey, error) {
	result, err := c.metaAPI.GetDeferredForeignKey(ctx, id)
	if err != nil {
		return nil, err
	}

	return newForeignKey(result)
}

// LookupForeignKey by source, target and fields
func (c *Client) LookupForeignKey(ctx context.Context, source string, target string, srcFields []string, dstFields []string) (*ForeignKey, error) {
	dataset, err := c.GetDataset(ctx, source)
	if err != nil {
		return nil, err
	}

	var matched *ForeignKeyConfig

	for _, fk := range dataset.ForeignKeys {
		switch {
		case fk.Target == nil || *fk.Target != target:
			continue
		case !reflect.DeepEqual(fk.SrcFields, srcFields):
			continue
		case !reflect.DeepEqual(fk.DstFields, dstFields):
			continue
		default:
			matched = &fk
			break
		}
	}

	if matched == nil {
		return nil, ErrNotFound
	}

	matched.Source = &dataset.ID

	return &ForeignKey{
		Workspace: dataset.WorkspaceID,
		Config:    matched,
	}, nil
}

// DeleteForeignKey
func (c *Client) DeleteForeignKey(ctx context.Context, id string) error {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.metaAPI.DeleteDeferredForeignKey(ctx, id)
}

// GetBookmarkGroup returns bookmarkGroup by ID
func (c *Client) GetBookmarkGroup(ctx context.Context, id string) (*BookmarkGroup, error) {
	result, err := c.metaAPI.GetBookmarkGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// CreateBookmarkGroup creates a bookmark group
func (c *Client) CreateBookmarkGroup(ctx context.Context, workspaceId string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	bookmarkGroupInput.WorkspaceID = toObjectPointer(&workspaceId)
	result, err := c.metaAPI.CreateOrUpdateBookmarkGroup(ctx, nil, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// UpdateBookmarkGroup updates a bookmark group
func (c *Client) UpdateBookmarkGroup(ctx context.Context, id string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmarkGroup(ctx, &id, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// DeleteBookmarkGroup
func (c *Client) DeleteBookmarkGroup(ctx context.Context, id string) error {
	return c.metaAPI.DeleteBookmarkGroup(ctx, id)
}

// GetBookmark returns bookmark by ID
func (c *Client) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	result, err := c.metaAPI.GetBookmark(ctx, id)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// CreateBookmark creates a bookmark group
func (c *Client) CreateBookmark(ctx context.Context, config *BookmarkConfig) (*Bookmark, error) {
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmark(ctx, nil, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// UpdateBookmark updates a bookmark
func (c *Client) UpdateBookmark(ctx context.Context, id string, config *BookmarkConfig) (*Bookmark, error) {
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmark(ctx, &id, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// DeleteBookmark
func (c *Client) DeleteBookmark(ctx context.Context, id string) error {
	return c.metaAPI.DeleteBookmark(ctx, id)
}
