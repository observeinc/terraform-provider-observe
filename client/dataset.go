package client

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
)

var (
	ErrDatasetNotFound = errors.New("dataset not found")
)

var FieldTypes = []string{
	"array",
	"bool",
	"float64",
	"int64",
	"object",
	"string",
	"timestamp",
}

// Dataset is published within a workspace, is the output of a transform.
type Dataset struct {
	WorkspaceID string        `json:"workspaceId"`
	ID          string        `json:"id"`
	Config      DatasetConfig `json:"config"`
}

// DatasetConfig declares configuration options for the Dataset. Use pointers to denote optional fields
type DatasetConfig struct {
	ID               string         `json:"id,omitempty"` // XXX: this should be part of Dataset, not properties
	Label            string         `json:"label,omitempty"`
	FreshnessDesired *time.Duration `json:"freshnessDesired,omitempty"`
	IconURL          *string        `json:"iconUrl,omitempty"`
}

type backendDatasetConfig struct {
	ID               string `json:"id,omitempty"`
	Label            string `json:"label,omitempty"`
	FreshnessDesired string `json:"freshnessDesired,omitempty"`
	IconURL          string `json:"iconUrl,omitempty"`
}

type backendDataset struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`

	Label            string `json:"label"`
	FreshnessDesired string `json:"freshnessDesired"`
	IconURL          string `json:"iconUrl"`
}

var (
	backendDatasetFragment = `
	fragment datasetFields on Dataset {
		workspaceId
		id
		label
		freshnessDesired
		iconUrl
	}`
	saveDatasetQuery = `
	mutation SaveDataset($workspaceId: ObjectId!, $dataset: DatasetInput!, $transform: TransformInput!) {
		saveDataset(
			workspaceId:$workspaceId
			dataset: $dataset
			transform: $transform
		) {
			dataset {
				...datasetFields
			}
		}
	}`
)

func (d *Dataset) fromBackend(b *backendDataset) error {
	d.WorkspaceID = b.WorkspaceID
	d.ID = b.ID
	d.Config.Label = b.Label

	if b.IconURL != "" {
		d.Config.IconURL = &b.IconURL
	}

	if b.FreshnessDesired != "" {
		i, err := strconv.Atoi(b.FreshnessDesired)
		if err != nil {
			return fmt.Errorf("could not convert freshness: %w", err)
		}
		freshness := time.Duration(int64(i))
		d.Config.FreshnessDesired = &freshness
	}

	return nil
}

func (c *DatasetConfig) toBackend(id string) *backendDatasetConfig {
	var b backendDatasetConfig
	if err := mapstructure.WeakDecode(c, &b); err != nil {
		panic(err)
	}

	b.ID = id
	return &b
}

func (c *Client) CreateDataset(workspaceID string, config DatasetConfig) (*Dataset, error) {

	// XXX: need a placeholder for now, just create a stage from observation table
	dataset, err := c.LookupDataset(workspaceID, "Observation")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup observation table: %w", err)
	}

	transformConfig, err := NewTransformConfig(nil, nil, &Stage{Input: dataset.ID, Pipeline: "filter true"})
	if err != nil {
		return nil, fmt.Errorf("failed to create transform config: %w", err)
	}

	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     config.toBackend(""),
		"transform":   transformConfig.toBackend(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	var b backendDataset
	if err := decode(getNested(result, "saveDataset", "dataset"), &b); err != nil {
		return nil, err
	}

	var d Dataset
	return &d, d.fromBackend(&b)
}

func (c *Client) UpdateDataset(workspaceID string, ID string, config DatasetConfig) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     config.toBackend(ID),
	})

	if err != nil {
		return nil, err
	}

	var b backendDataset
	if err := decode(getNested(result, "saveDataset", "dataset"), &b); err != nil {
		return nil, err
	}

	var d Dataset
	return &d, d.fromBackend(&b)
}

func (c *Client) LookupDataset(workspaceID string, label string) (*Dataset, error) {
	// TODO: we need an endpoint to lookup dataset by label
	// For now be lazy and reuse list function

	datasets, err := c.ListDatasets()
	if err != nil {
		return nil, err
	}

	for _, d := range datasets {
		if d.WorkspaceID == workspaceID && d.Config.Label == label {
			return d, nil
		}
	}

	return nil, ErrDatasetNotFound
}

func (c *Client) GetDataset(id string) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+`
	query GetDataset($id: ObjectId!) {
		dataset(id:$id) {
			...datasetFields
		}
	}`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return nil, err
	}

	var b backendDataset
	if err := decode(getNested(result, "dataset"), &b); err != nil {
		return nil, err
	}

	var d Dataset
	return &d, d.fromBackend(&b)
}

// ListDatasets retrieves all datasets across workspaces. No filtering provided for now.
func (c *Client) ListDatasets() (ds []*Dataset, err error) {
	result, err := c.Run(backendDatasetFragment+`
	query {
		projects {
			datasets {
				...datasetFields
			}
		}
	}`, nil)

	if err != nil {
		return nil, err
	}

	for _, elem := range result["projects"].([]interface{}) {
		var bs []backendDataset
		if err := decode(getNested(elem, "datasets"), &bs); err != nil {
			return nil, err
		}

		for _, b := range bs {
			var d Dataset
			if err := d.fromBackend(&b); err != nil {
				return nil, fmt.Errorf("failed to convert dataset: %w", err)
			}
			ds = append(ds, &d)
		}
	}
	return ds, nil
}

// DeleteDataset deletes dataset by ID.
func (c *Client) DeleteDataset(id string) error {
	result, err := c.Run(`
	mutation ($id: ObjectId!) {
		deleteDataset(dsid: $id) {
			success
			errorMessage
		}
	}`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	if err := mapstructure.Decode(result["deleteDataset"], &status); err != nil {
		return err
	}

	return status.Error()
}
