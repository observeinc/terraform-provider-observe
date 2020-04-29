package client

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")
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
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.api.CreateDeferredForeignKey(workspaceID, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		c.DeleteForeignKey(result.ID.String())
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// UpdateForeignKey by ID
func (c *Client) UpdateForeignKey(id string, config *ForeignKeyConfig) (*ForeignKey, error) {
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

// DeleteForeignKey
func (c *Client) DeleteForeignKey(id string) error {
	return c.api.DeleteDeferredForeignKey(id)
}
