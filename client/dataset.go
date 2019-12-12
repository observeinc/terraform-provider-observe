package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/mitchellh/mapstructure"
)

var (
	ErrDatasetNotFound = errors.New("dataset not found")
)

type backendInput struct {
	InputName string `json:"inputName"`
	StageID   string `json:"stageID,omitempty"`
	DatasetID string `json:"datasetId,omitempty"`
}

type backendStage struct {
	StageID  string         `json:"stageID"`
	Input    []backendInput `json:"input"`
	Pipeline string         `json:"pipeline"`
}

type backendDataset struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	WorkspaceID string `json:"workspaceId"`
	Transform   struct {
		Current struct {
			Stages []backendStage `json:"stages"`
		} `json:"current"`
	} `json:"transform"`
}

var (
	backendDatasetFragment = `
	fragment datasetFields on Dataset {
		id
		label
		workspaceId
		transform {
			current {
				stages {
					stageID
					pipeline
					input {
						inputName
						datasetId
					}
				}
			}
		}
	}`
	saveDatasetQuery = `
	mutation SaveDataset($workspaceId: ObjectId!, $datasetId: ObjectId, $label: String!, $transform: TransformInput!) {
		saveDataset(
			workspaceId:$workspaceId
			dataset: {
				id: $datasetId
				label: $label
				deleted: false
			}
			transform: $transform
		) {
			dataset {
				...datasetFields
			}
		}
	}`
)

func getNested(i interface{}, keys ...string) interface{} {
	for _, k := range keys {
		v, ok := i.(map[string]interface{})
		if !ok {
			return nil
		}
		i = v[k]
	}
	return i
}

type Dataset struct {
	WorkspaceID string     `json:"workspaceId"`
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Transform   *Transform `json:"transform"`
}

func newDataset(input interface{}) (*Dataset, error) {
	if input == nil {
		return nil, ErrDatasetNotFound
	}

	var result backendDataset
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &result,
		ErrorUnused: true,
	})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	var inputs []Input
	var stages []Stage
	for _, s := range result.Transform.Current.Stages {
		stages = append(stages, Stage{
			Name:     s.StageID,
			Pipeline: NewPipeline(s.Pipeline),
		})

		if len(inputs) == 0 {
			for _, i := range s.Input {
				inputs = append(inputs, Input{
					Name:      i.InputName,
					DatasetID: i.DatasetID,
				})
			}
		}
	}

	dataset := &Dataset{
		WorkspaceID: result.WorkspaceID,
		ID:          result.ID,
		Label:       result.Label,
		Transform: &Transform{
			Inputs: inputs,
			Stages: stages,
		},
	}

	return dataset, nil
}

// Transform is simplified - we support only one stage
type Transform struct {
	Inputs []Input `json:"inputs"`
	Stages []Stage `json:"stages"`
}

type DatasetInput struct {
	WorkspaceID string  `json:"workspaceId"`
	Label       string  `json:"label"`
	Inputs      []Input `json:"inputs"`
	Stages      []Stage `json:"stages"`
}

func (d *DatasetInput) transform() interface{} {
	var inputs []backendInput
	for _, i := range d.Inputs {
		inputs = append(inputs, backendInput{
			InputName: i.Name,
			DatasetID: i.DatasetID,
		})
	}

	var stages []backendStage

	for _, s := range d.Stages {
		currentInputs := inputs
		stages = append(stages, backendStage{
			StageID:  s.Name,
			Pipeline: s.Pipeline.Canonical(),
			Input:    currentInputs,
		})

		inputs = append(inputs, backendInput{
			InputName: s.Name,
			StageID:   s.Name,
		})
	}

	return struct {
		OutputStage string         `json:"outputStage"`
		Stages      []backendStage `json:"stages"`
	}{
		OutputStage: stages[len(stages)-1].StageID,
		Stages:      stages,
	}
}

type Input struct {
	Name      string `json:"name"`
	DatasetID string `json:"datasetId"`
}

type Stage struct {
	Name     string    `json:"name"`
	Pipeline *Pipeline `json:"pipeline"`
}

func (c *Client) CreateDataset(input DatasetInput) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"workspaceId": input.WorkspaceID,
		"label":       input.Label,
		"transform":   input.transform(),
	})

	if err != nil {
		return nil, err
	}

	return newDataset(getNested(result, "saveDataset", "dataset"))
}

func (c *Client) UpdateDataset(id string, input DatasetInput) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"datasetId":   id,
		"workspaceId": input.WorkspaceID,
		"label":       input.Label,
		"transform":   input.transform(),
	})

	if err != nil {
		return nil, err
	}

	return newDataset(getNested(result, "saveDataset", "dataset"))
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

	return newDataset(getNested(result, "dataset"))
}

func (c *Client) ListDatasets() ([]*Dataset, error) {
	return nil, fmt.Errorf("nope")
}

/*
	workspaces := []struct {
		ID       string `json:"id"`
		Datasets []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"datasets"`
	}{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &workspaces,
	})

	for _, w := range workspaces {
		for _, d := range w.Datasets {
			datasets = append(datasets, &Dataset{})
		}
	}
}

/*
	client, err := sharedClient()
	if err != nil {
		t.Fatalf("failed to load client:", err)
	}



	if err := decoder.Decode(result["projects"]); err != nil {
		t.Fatal(err)
	}

*/

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
