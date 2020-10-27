package client

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNotFound = errors.New("not found")

	flagObs2110 = "obs2110" // when set, allow concurrent API calls for foreign keys
)

// GetDataset returns dataset by ID
func (c *Client) GetDataset(id string) (*Dataset, error) {
	result, err := c.metaAPI.GetDataset(id)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// CreateDataset creates dataset
func (c *Client) CreateDataset(workspaceId string, config *DatasetConfig) (*Dataset, error) {
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.SaveDataset(workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// UpdateDataset updates existing dataset
func (c *Client) UpdateDataset(workspaceId string, id string, config *DatasetConfig) (*Dataset, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	datasetInput.ID = toObjectPointer(&id)
	result, err := c.metaAPI.SaveDataset(workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// DeleteDataset by ID
func (c *Client) DeleteDataset(id string) error {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.metaAPI.DeleteDataset(id)
}

// GetWorkspace by ID
func (c *Client) GetWorkspace(id string) (*Workspace, error) {
	result, err := c.metaAPI.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return newWorkspace(result)
}

// LookupWorkspace by name.
func (c *Client) LookupWorkspace(name string) (*Workspace, error) {
	workspaces, err := c.metaAPI.ListWorkspaces()
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
func (c *Client) ListWorkspaces() (workspaces []*Workspace, err error) {
	result, err := c.metaAPI.ListWorkspaces()
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
func (c *Client) LookupDataset(workspaceID string, name string) (*Dataset, error) {
	workspace, err := c.GetWorkspace(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup dataset: %w", err)
	}

	id, ok := workspace.Datasets[name]
	if !ok {
		return nil, ErrNotFound
	}
	return c.GetDataset(id)
}

// CreateForeignKey
func (c *Client) CreateForeignKey(workspaceID string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.CreateDeferredForeignKey(workspaceID, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		// call internal API directly since DeleteForeignKey() acquires lock
		c.metaAPI.DeleteDeferredForeignKey(result.ID.String())
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// UpdateForeignKey by ID
func (c *Client) UpdateForeignKey(id string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.metaAPI.UpdateDeferredForeignKey(id, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// GetForeignKey returns deferred foreign key
func (c *Client) GetForeignKey(id string) (*ForeignKey, error) {
	result, err := c.metaAPI.GetDeferredForeignKey(id)
	if err != nil {
		return nil, err
	}

	return newForeignKey(result)
}

// LookupForeignKey by source, target and fields
func (c *Client) LookupForeignKey(source string, target string, srcFields []string, dstFields []string) (*ForeignKey, error) {
	dataset, err := c.GetDataset(source)
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
func (c *Client) DeleteForeignKey(id string) error {
	if !c.flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.metaAPI.DeleteDeferredForeignKey(id)
}

// GetBookmarkGroup returns bookmarkGroup by ID
func (c *Client) GetBookmarkGroup(id string) (*BookmarkGroup, error) {
	result, err := c.metaAPI.GetBookmarkGroup(id)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// CreateBookmarkGroup creates a bookmark group
func (c *Client) CreateBookmarkGroup(workspaceId string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	bookmarkGroupInput.WorkspaceID = toObjectPointer(&workspaceId)
	result, err := c.metaAPI.CreateOrUpdateBookmarkGroup(nil, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// UpdateBookmarkGroup updates a bookmark group
func (c *Client) UpdateBookmarkGroup(id string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmarkGroup(&id, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// DeleteBookmarkGroup
func (c *Client) DeleteBookmarkGroup(id string) error {
	return c.metaAPI.DeleteBookmarkGroup(id)
}

// GetBookmark returns bookmark by ID
func (c *Client) GetBookmark(id string) (*Bookmark, error) {
	result, err := c.metaAPI.GetBookmark(id)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// CreateBookmark creates a bookmark group
func (c *Client) CreateBookmark(config *BookmarkConfig) (*Bookmark, error) {
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmark(nil, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// UpdateBookmark updates a bookmark
func (c *Client) UpdateBookmark(id string, config *BookmarkConfig) (*Bookmark, error) {
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.metaAPI.CreateOrUpdateBookmark(&id, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// DeleteBookmark
func (c *Client) DeleteBookmark(id string) error {
	return c.metaAPI.DeleteBookmark(id)
}
