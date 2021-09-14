package meta

import (
	"context"
)

var (
	backendDatasetFragment = `
	fragment datasetFields on Dataset {
		workspaceId
		id
		label
		freshnessDesired
		description
		iconUrl
		version
		lastSaved
		pathCost
		source
		foreignKeys {
			label
			targetDataset
			srcFields
			dstFields
		}
		transform {
			current {
				query {
					outputStage
					stages {
						id
						pipeline
						input {
							inputName
							inputRole
							datasetId
							datasetPath
							stageId
						}
					}
				}
			}
		}
		typedef {
			definition
		}
		sourceTable {
			schema
			tableName
			sourceUpdateTableName
			isInsertOnly
			batchSeqField
			validFromField
			fields {
				name
				sqlType
			}
		}
	}`
)

// SaveDataset creates and updates datasets
func (c *Client) SaveDataset(ctx context.Context, workspaceID string, d *DatasetInput, q *MultiStageQueryInput) (*Dataset, error) {
	result, err := c.Run(ctx, backendDatasetFragment+`
	mutation saveDataset($workspaceId: ObjectId!, $dataset: DatasetInput!, $query: MultiStageQueryInput!, $dep: DependencyHandlingInput) {
		saveDataset(workspaceId:$workspaceId, dataset:$dataset, query:$query, dependencyHandling:$dep) {
			dataset {
				...datasetFields
			}
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     d,
		"query":       q,
		"dep":         &DependencyHandlingInput{SaveMode: SaveModeUpdateDataset},
	})

	if err != nil {
		return nil, err
	}

	var ds DatasetSaveResult
	err = decodeStrict(getNested(result, "saveDataset"), &ds)
	return ds.Dataset, err
}

func (c *Client) SaveSourceDataset(ctx context.Context, workspaceID string, d *DatasetDefinitionInput, s *SourceTableDefinitionInput) (*Dataset, error) {
	result, err := c.Run(ctx, backendDatasetFragment+`
	mutation saveSourceDataset($workspaceId: ObjectId!, $datasetDefinition: DatasetDefinitionInput!, $sourceTable: SourceTableDefinitionInput!, $dep: DependencyHandlingInput) {
		saveSourceDataset(workspaceId:$workspaceId, datasetDefinition:$datasetDefinition, sourceTable:$sourceTable, dependencyHandling:$dep) {
			dataset {
				...datasetFields
			}
		}
	}`, map[string]interface{}{
		"workspaceId":       workspaceID,
		"datasetDefinition": d,
		"sourceTable":       s,
		"dep":               &DependencyHandlingInput{SaveMode: SaveModeUpdateDataset},
	})

	if err != nil {
		return nil, err
	}

	var ds DatasetSaveResult
	err = decodeStrict(getNested(result, "saveSourceDataset"), &ds)
	return ds.Dataset, err
}

// GetDataset retrieves dataset.
func (c *Client) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	result, err := c.Run(ctx, backendDatasetFragment+`
	query getDataset($id: ObjectId!) {
        dataset(id: $id) {
            ...datasetFields
        }
    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return nil, err
	}

	var dataset Dataset
	if err := decodeStrict(getNested(result, "dataset"), &dataset); err != nil {
		return nil, err
	}
	return &dataset, nil
}

// LookupDataset retrieves dataset by name.
func (c *Client) LookupDataset(ctx context.Context, workspaceId, name string) (*Dataset, error) {
	result, err := c.Run(ctx, backendDatasetFragment+`
	query lookupDataset($workspaceId: ObjectId!, $name: String!) {
		workspace(id: $workspaceId) {
			dataset(label: $name) {
            	...datasetFields
        	}
		}
    }`, map[string]interface{}{
		"workspaceId": workspaceId,
		"name":        name,
	})

	if err != nil {
		return nil, err
	}

	var dataset Dataset
	if err := decodeStrict(getNested(result, "workspace", "dataset"), &dataset); err != nil {
		return nil, err
	}
	return &dataset, nil
}

// DeleteDataset deletes dataset by ID.
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!,  $dep: DependencyHandlingInput) {
        deleteDataset(dsid: $id, dependencyHandling:$dep) {
            success
            errorMessage
            detailedInfo
        }
    }`, map[string]interface{}{
		"id":  id,
		"dep": &DependencyHandlingInput{SaveMode: SaveModeUpdateDataset},
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

// ListDatasets retrieves all datasets across workspaces. No filtering provided for now.
func (c *Client) ListDatasets(ctx context.Context) (ds []*Dataset, err error) {
	result, err := c.Run(ctx, backendDatasetFragment+`
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

	var projects struct {
		Projects []*Workspace `json:"projects"`
	}
	if err := decodeStrict(result, &projects); err != nil {
		return nil, err
	}

	var datasets []*Dataset
	for _, workspace := range projects.Projects {
		datasets = append(datasets, workspace.Datasets...)
	}

	return datasets, nil
}
