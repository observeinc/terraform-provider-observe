package client

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrDatasetNotFound = errors.New("dataset not found")

	backendDatasetFragment = `
	fragment datasetFields on Dataset {
		workspaceId
		id
		label
		freshnessDesired
		iconUrl
		typedef {
		  definition
		}
		validFromField
		validToField
		labelField
	}`

	defineDatasetQuery = `
	mutation DefineDataset($workspaceId: ObjectId!, $definition: DatasetDefinitionInput!) {
		defineDataset(
			workspaceId:$workspaceId
			definition: $definition
		) {
			...datasetFields
		}
	}`
)

// Dataset is published within a workspace
type Dataset struct {
	WorkspaceID string        `json:"workspaceId"`
	ID          string        `json:"id"`
	Config      DatasetConfig `json:"config"`
}

// DatasetConfig contains all the configurable elements of a dataset.
type DatasetConfig struct {
	Name             string         `json:"name,omitempty"`
	FreshnessDesired *time.Duration `json:"freshnessDesired,omitempty"`
	IconURL          *string        `json:"iconUrl,omitempty"`

	// schema
	Fields []*Field `json:"fields,omitempty"`
}

type Field struct {
	Name string `json:"field,omitempty"`
	Type string `json:"type,omitempty"`

	ValidTo   bool `json:"valid_to,omitempty"`
	ValidFrom bool `json:"valid_from,omitempty"`
	Label     bool `json:"label,omitempty"`
}

var FieldTypes = []string{
	"array",
	"bool",
	"float64",
	"int64",
	"object",
	"string",
	"timestamp",
}

type fieldType struct {
	Rep      string `json:"rep"`
	Nullable *bool  `json:"nullable,omitempty"`
}

type fieldDef struct {
	Name         string     `json:"name,omitempty"`
	IsConst      *bool      `json:"isConst,omitempty"`
	IsEnum       *bool      `json:"isEnum,omitempty"`
	IsHidden     *bool      `json:"isHidden,omitempty"`
	IsSearchable *bool      `json:"isSearchable,omitempty"`
	Label        *string    `json:"label,omitempty"`
	Type         *fieldType `json:"type"`
}

func (f *Field) toBackend() interface{} {
	return &fieldDef{
		Name:  f.Name,
		Label: &f.Name,
		Type: &fieldType{
			Rep: f.Type,
		},
	}
}

func (f *Field) fromBackend(data interface{}) error {
	var backend fieldDef
	if err := decodeLoose(data, &backend); err != nil {
		return err
	}

	f.Name = backend.Name
	f.Type = backend.Type.Rep
	return nil
}

func (d *Dataset) fromBackend(data interface{}) error {
	type Dataset struct {
		ID          string `json:"id"`
		WorkspaceID string `json:"workspaceId"`

		Label            string `json:"label"`
		FreshnessDesired string `json:"freshnessDesired"`
		IconURL          string `json:"iconUrl"`
		Typedef          struct {
			Definition struct {
				Fields []interface{} `json:"fields,omitempty"`
			} `json:"definition,omitempty"`
		} `json:"typedef,omitempty"`
		ValidFromField *string `json:"validFromField"`
		ValidToField   *string `json:"validToField"`
		LabelField     *string `json:"labelField"`
	}

	var backend Dataset
	if err := decodeStrict(data, &backend); err != nil {
		return err
	}

	d.WorkspaceID = backend.WorkspaceID
	d.ID = backend.ID
	d.Config.Name = backend.Label

	if backend.IconURL != "" {
		d.Config.IconURL = &backend.IconURL
	}

	if backend.FreshnessDesired != "" {
		i, err := strconv.Atoi(backend.FreshnessDesired)
		if err != nil {
			return fmt.Errorf("could not convert freshness: %w", err)
		}
		freshness := time.Duration(int64(i))
		d.Config.FreshnessDesired = &freshness
	}

	for _, f := range backend.Typedef.Definition.Fields {
		var field Field
		if err := field.fromBackend(f); err != nil {
			return fmt.Errorf("failed to decode field: %w", err)
		}

		if backend.ValidFromField != nil && *backend.ValidFromField == field.Name {
			field.ValidFrom = true
		}

		if backend.ValidToField != nil && *backend.ValidToField == field.Name {
			field.ValidTo = true
		}

		if backend.LabelField != nil && *backend.LabelField == field.Name {
			field.Label = true
		}

		d.Config.Fields = append(d.Config.Fields, &field)
	}

	return nil
}

func (c *DatasetConfig) toDatasetDefinition(id string) interface{} {
	type DatasetInput struct {
		ID               string `json:"id,omitempty"`
		Label            string `json:"label,omitempty"`
		FreshnessDesired string `json:"freshnessDesired,omitempty"`
		IconURL          string `json:"iconUrl,omitempty"`
	}

	type DatasetDefinition struct {
		Dataset DatasetInput  `json:"dataset,omitempty"`
		Schema  []interface{} `json:"schema,omitempty"`
		//Metadata *backendDefinitionMetadataInput `json:"metadata,omitempty"`
	}

	backend := &DatasetDefinition{
		Dataset: DatasetInput{
			ID:    id,
			Label: c.Name,
		},
	}

	if c.FreshnessDesired != nil {
		backend.Dataset.FreshnessDesired = fmt.Sprintf("%d", c.FreshnessDesired.Nanoseconds())
	}
	if c.IconURL != nil {
		backend.Dataset.IconURL = *c.IconURL
	}
	for _, f := range c.Fields {
		backend.Schema = append(backend.Schema, f.toBackend())
	}

	return &backend
}

func (c *Client) CreateDataset(workspaceID string, config DatasetConfig) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+defineDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"definition":  config.toDatasetDefinition(""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	var d Dataset
	return &d, d.fromBackend(getNested(result, "defineDataset"))
}

func (c *Client) UpdateDataset(workspaceID string, ID string, config DatasetConfig) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+defineDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"definition":  config.toDatasetDefinition(ID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update dataset: %w", err)
	}

	var d Dataset
	return &d, d.fromBackend(getNested(result, "defineDataset"))
}

func (c *Client) LookupDataset(workspaceID string, label string) (*Dataset, error) {
	// TODO: we need an endpoint to lookup dataset by label
	// For now be lazy and reuse list function

	datasets, err := c.ListDatasets()
	if err != nil {
		return nil, err
	}

	for _, d := range datasets {
		if d.WorkspaceID == workspaceID && d.Config.Name == label {
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

	var d Dataset
	value := getNested(result, "dataset")
	if value == nil {
		return nil, nil
	}
	return &d, d.fromBackend(value)
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
		var bs []map[string]interface{}
		nested := getNested(elem, "datasets")
		if err := decodeStrict(nested, &bs); err != nil {
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
	nested := getNested(result, "deleteDataset")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}

	return status.Error()
}
