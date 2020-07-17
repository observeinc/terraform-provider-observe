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
	result, err := c.api.GetDataset(id)
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
	result, err := c.api.SaveDataset(workspaceId, datasetInput, transformInput)
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
	result, err := c.api.SaveDataset(workspaceId, datasetInput, transformInput)
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
	return c.api.DeleteDataset(id)
}

// GetWorkspace by ID
func (c *Client) GetWorkspace(id string) (*Workspace, error) {
	result, err := c.api.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return newWorkspace(result)
}

// LookupWorkspace by name.
func (c *Client) LookupWorkspace(name string) (*Workspace, error) {
	workspaces, err := c.api.ListWorkspaces()
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
	result, err := c.api.ListWorkspaces()
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
	result, err := c.api.CreateDeferredForeignKey(workspaceID, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		// call internal API directly since DeleteForeignKey() acquires lock
		c.api.DeleteDeferredForeignKey(result.ID.String())
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
	result, err := c.api.UpdateDeferredForeignKey(id, foreignKeyInput)
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
	result, err := c.api.GetDeferredForeignKey(id)
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
	return c.api.DeleteDeferredForeignKey(id)
}
