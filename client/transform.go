package client

import (
	"encoding/json"
	"fmt"
	"log"
)

// TransformConfig describes a sequence of stages
type TransformConfig struct {
	Inputs        map[string]string `json:"inputs"`
	References    map[string]string `json:"references"`
	Stages        []*Stage          `json:"stages"`
	inputs        map[string]*backendInput
	backendStages []*backendStage
}

type Transform struct {
	ID string `json:"id"`
	*TransformConfig
}

// Stage declares a source to operate on, and a pipeline to execute
type Stage struct {
	Name     string `json:"name,omitempty"`
	Input    string `json:"input,omitempty"`
	Pipeline string `json:"pipeline,omitempty"`
}

type backendInput struct {
	InputName string `json:"inputName"`
	StageID   string `json:"stageID,omitempty"`
	DatasetID string `json:"datasetId,omitempty"`
	InputRole string `json:"inputRole,omitempty"`
}

type backendStage struct {
	Input    []*backendInput `json:"input"`
	StageID  string          `json:"stageID"`
	Pipeline string          `json:"pipeline"`
}

type backendTransform struct {
	OutputStage string          `json:"outputStage"`
	Stages      []*backendStage `json:"stages"`
}

type backendDatasetWithTransform struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`

	Label            string `json:"label"`
	FreshnessDesired string `json:"freshnessDesired"`
	IconURL          string `json:"iconUrl"`

	Transform struct {
		Current backendTransform `json:"current"`
	} `json:"transform"`
}

var (
	backendTransformFragment = `
	fragment datasetFields on Dataset {
		id
		label
		workspaceId
		freshnessDesired
		iconUrl
		transform {
			id
			current {
				outputStage
				stages {
					stageID
					pipeline
					input {
						inputName
						datasetId
						stageID
					}
				}
			}
		}
	}`
	saveTransformQuery = `
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

func NewTransformConfig(inputs map[string]string, references map[string]string, stages ...*Stage) (*TransformConfig, error) {
	t := &TransformConfig{
		Inputs:     inputs,
		References: references,
		Stages:     stages,
	}

	t.inputs = make(map[string]*backendInput)
	for k, v := range t.Inputs {
		t.inputs[k] = &backendInput{InputName: k, DatasetID: v}
	}
	for k, v := range t.References {
		t.inputs[k] = &backendInput{InputName: k, DatasetID: v, InputRole: "Reference"}
	}

	for i, s := range stages {
		var stageName string
		if stageName = s.Name; stageName == "" {
			stageName = fmt.Sprintf("stage%d", i)
		}

		// validate stage definition
		var err error
		switch {
		case i == 0 && s.Input == "":
			err = fmt.Errorf("first stage must declare an import")
		case s.Pipeline == "":
			err = fmt.Errorf("stage must declare a pipeline")
		case stageName != "" && t.inputs[stageName] != nil:
			err = fmt.Errorf("stage %s already declared", stageName)
		}
		if err != nil {
			return nil, err
		}

		defaultBinding := s.Input
		if defaultBinding == "" {
			defaultBinding = t.backendStages[i-1].StageID
		}

		t.backendStages = append(t.backendStages, &backendStage{
			StageID:  stageName,
			Pipeline: NewPipeline(s.Pipeline).Canonical(),
			Input:    t.constructStageInputs(defaultBinding),
		})

		t.inputs[stageName] = &backendInput{
			InputName: stageName,
			StageID:   stageName,
		}
	}

	if len(t.Stages) != len(t.backendStages) {
		panic("wrong number of stages")
	}

	return t, nil
}

func (t *TransformConfig) constructStageInputs(defaultBinding string) (inputs []*backendInput) {
	if input, ok := t.inputs[defaultBinding]; ok {
		inputs = append(inputs, input)
	} else {
		// allow direct references to a datasetID
		inputs = append(inputs, &backendInput{InputName: defaultBinding, DatasetID: defaultBinding})
	}

	for name, input := range t.inputs {
		if name != defaultBinding {
			inputs = append(inputs, input)
		}
	}
	return
}

func (t *TransformConfig) toBackend() *backendTransform {
	var outputStageName string

	if len(t.Stages) != len(t.backendStages) {
		panic("wrong number of stages")
	}
	if len(t.backendStages) > 0 {
		outputStageName = t.backendStages[len(t.backendStages)-1].StageID
	}

	return &backendTransform{
		OutputStage: outputStageName,
		Stages:      t.backendStages,
	}
}

func (t *TransformConfig) fromBackend(b *backendTransform) error {
	t.References = make(map[string]string)
	t.Inputs = make(map[string]string)

	t.backendStages = b.Stages

	s, _ := json.Marshal(b.Stages)
	log.Printf("backend stages: %s\n", string(s))

	for i, backendStage := range b.Stages {
		var s Stage

		if backendStage.StageID != fmt.Sprintf("stage%d", i) {
			s.Name = backendStage.StageID
		}

		s.Pipeline = NewPipeline(backendStage.Pipeline).String()

		defaultInput := backendStage.Input[0]

		switch {
		case defaultInput.InputName == defaultInput.DatasetID:
			// direct dataset reference
			s.Input = defaultInput.InputName
		case defaultInput.InputName == defaultInput.StageID && defaultInput.StageID == b.Stages[i-1].StageID:
			// default follow on case
		default:
			s.Input = defaultInput.InputName
		}

		t.Stages = append(t.Stages, &s)

		for _, input := range backendStage.Input {
			switch {
			case input.StageID != "":
			case input.InputRole == "Reference":
				t.References[input.InputName] = input.DatasetID
			default:
				t.Inputs[input.InputName] = input.DatasetID
			}
		}
	}

	if len(t.Stages) != len(t.backendStages) {
		panic("wrong number of stages")
	}

	return nil
}

func (c *Client) SetTransform(datasetID string, config *TransformConfig) (*Transform, error) {
	dataset, err := c.GetDataset(datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dataset: %w", err)
	}

	if config == nil {
		// delete by resetting transform to something else
		dataset, err := c.LookupDataset(dataset.WorkspaceID, "Observation")
		if err != nil {
			return nil, fmt.Errorf("failed to lookup observation table: %w", err)
		}
		config, err = NewTransformConfig(nil, nil, &Stage{Input: dataset.ID, Pipeline: "filter true"})
		if err != nil {
			return nil, fmt.Errorf("failed to create transform config: %w", err)
		}
	}

	result, err := c.Run(backendTransformFragment+saveTransformQuery, map[string]interface{}{
		"workspaceId": dataset.WorkspaceID,
		"dataset":     dataset.Config.toBackend(dataset.ID),
		"transform":   config.toBackend(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure transform: %w", err)
	}

	var b backendTransform
	if err := decode(getNested(result, "saveDataset", "dataset", "transform", "current"), &b); err != nil {
		return nil, err
	}

	var t TransformConfig
	if err := t.fromBackend(&b); err != nil {
		return nil, fmt.Errorf("failed to convert transform config: %w", err)
	}

	return &Transform{
		ID:              dataset.ID,
		TransformConfig: &t,
	}, nil
}

func (c *Client) GetTransform(id string) (*Transform, error) {
	result, err := c.Run(backendTransformFragment+`
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

	var b backendTransform
	if err := decode(getNested(result, "dataset", "transform", "current"), &b); err != nil {
		return nil, err
	}

	var t TransformConfig

	if err := t.fromBackend(&b); err != nil {
		return nil, fmt.Errorf("failed to convert transfrom: %w", err)
	}

	if len(t.Stages) == 0 {
		return nil, fmt.Errorf("no stages found for transform %s", id)
	}

	return &Transform{
		ID:              id,
		TransformConfig: &t,
	}, nil
}
