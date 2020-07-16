package api

var (
	backendDatasetFragment = `
	fragment datasetFields on Dataset {
		workspaceId
		id
		label
		freshnessDesired
		iconUrl
		version
		pathCost
		foreignKeys {
			label
			targetDataset
			srcFields
			dstFields
		}
		transform {
			current {
				outputStage
				stages {
					stageID
					pipeline
					input {
						inputName
						inputRole
						datasetId
						datasetPath
						stageID
					}
				}
			}
		}
	}`
)

// SaveDataset creates and updates datasets
func (c *Client) SaveDataset(workspaceID string, d *DatasetInput, t *TransformInput) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+`
	mutation saveDataset($workspaceId: ObjectId!, $dataset: DatasetInput!, $transform: TransformInput!, $dep: DependencyHandlingInput) {
		saveDataset(workspaceId:$workspaceId, dataset:$dataset, transform:$transform, dependencyHandling:$dep) {
			dataset {
				...datasetFields
			}
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     d,
		"transform":   t,
		"dep":         &DependencyHandlingInput{SaveMode: SaveModeUpdateDataset},
	})

	if err != nil {
		return nil, err
	}

	var ds DatasetSaveResult
	err = decodeStrict(getNested(result, "saveDataset"), &ds)
	return ds.Dataset, err
}

// GetDataset retrieves dataset.
func (c *Client) GetDataset(id string) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+`
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

// DeleteDataset deletes dataset by ID.
func (c *Client) DeleteDataset(id string) error {
	result, err := c.Run(`
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
