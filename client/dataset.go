package client

import (
	"context"
	//"encoding/json"
	"errors"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/mitchellh/mapstructure"
)

var (
	ErrDatasetNotFound = errors.New("dataset not found")
)

type Dataset struct {
	WorkspaceID string     `json:"workspaceId"`
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Transform   *Transform `json:"transform"`
}

func decodeDataset(input interface{}) (*Dataset, error) {
	if input == nil {
		return nil, ErrDatasetNotFound
	}

	backendDefinition := struct {
		Dataset struct {
			ID          string `json:"id"`
			Label       string `json:"label"`
			WorkspaceID string `json:"workspaceId"`
			Transform   struct {
				Current struct {
					Stages []struct {
						Pipeline string  `json:"pipeline"`
						Input    []Input `json:"input"`
					} `json:"stages"`
				} `json:"current"`
			} `json:"transform"`
		} `json:"dataset"`
	}{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &backendDefinition,
		ErrorUnused: true,
	})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	var t *Transform
	switch len(backendDefinition.Dataset.Transform.Current.Stages) {
	case 0:
	case 1:
		stage := backendDefinition.Dataset.Transform.Current.Stages[0]
		t = &Transform{
			Pipeline: NewPipeline(stage.Pipeline),
			Inputs:   stage.Input,
		}
	default:
		return nil, fmt.Errorf("unsupported transform, more than one stage defined")
	}

	dataset := &Dataset{
		WorkspaceID: backendDefinition.Dataset.WorkspaceID,
		ID:          backendDefinition.Dataset.ID,
		Label:       backendDefinition.Dataset.Label,
		Transform:   t,
	}

	return dataset, nil
}

// Transform is simplified - we support only one stage
type Transform struct {
	Inputs   []Input   `json:"inputs"`
	Pipeline *Pipeline `json:"pipeline"`
}

type CreateDatasetInput struct {
	WorkspaceID string    `json:"workspaceId"`
	Label       string    `json:"label"`
	Inputs      []Input   `json:"inputs"`
	Pipeline    *Pipeline `json:"pipeline"`
}

type Input struct {
	InputName string `json:"inputName"`
	DatasetID string `json:"datasetId"`
}

func (c *Client) CreateDataset(input CreateDatasetInput) (*Dataset, error) {
	result, err := c.Run(`
	mutation CreateDataset($workspaceId: ObjectId!, $datasetId: ObjectId, $label: String!, $pipeline: String!, $inputs: [InputDefinitionInput!]!) {
		saveDataset(
			workspaceId:$workspaceId
			dataset: {
				id: $datasetId
				label: $label
				deleted: false
			}
			transform: {
				outputStage: "0"
				stages: [{
					stageID: "0"
					pipeline: $pipeline
					input: $inputs
				}]
			}
		) {
			dataset {
				id
				workspaceId
				label
				transform {
					current {
						stages {
							pipeline
						}
					}
				}
			}
		}
	}`, map[string]interface{}{
		"workspaceId": input.WorkspaceID,
		"label":       input.Label,
		"pipeline":    input.Pipeline.Canonical(),
		"inputs":      input.Inputs,
	})

	if err != nil {
		return nil, err
	}

	return decodeDataset(result["saveDataset"])
}

func (c *Client) LookupDataset(workspaceID string, label string) (*Dataset, error) {
	// TODO: we need an endpoint to lookup dataset by label
	req := graphql.NewRequest(`
	query ($workspaceId: ObjectId!) {
		project(projectId:$workspaceId) {
			datasets {
				id
				label
			}
		}
	}`)

	req.Var("workspaceId", workspaceID)

	var respData struct {
		Project struct {
			Datasets []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"datasets"`
		} `json:"project"`
	}

	if err := c.client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	for _, d := range respData.Project.Datasets {
		if d.Label == label {
			return c.GetDataset(d.ID)
		}
	}

	return nil, ErrDatasetNotFound
}

func (c *Client) GetDataset(id string) (*Dataset, error) {

	result, err := c.Run(`
	query GetDataset($id: ObjectId!) {
		dataset(id:$id) {
			id
			label
			transform {
				current {
					stages {
						pipeline
						input {
							inputName
							datasetId
						}
					}
				}
			}
		}
	}`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return nil, err
	}

	return decodeDataset(result)
}

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
