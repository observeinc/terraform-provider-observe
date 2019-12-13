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

// Dataset is published within a workspace, is the output of a transform.
type Dataset struct {
	WorkspaceID string          `json:"workspaceId"`
	ID          string          `json:"id"`
	Config      DatasetConfig   `json:"config"`
	Transform   TransformConfig `json:"transform"`
}

// DatasetConfig declares configuration options for the Dataset. Use pointers to denote optional fields
type DatasetConfig struct {
	ID               string         `json:"id,omitempty"` // XXX: this should be part of Dataset, not properties
	Label            string         `json:"label,omitempty"`
	Deleted          *bool          `json:"deleted,omitempty"`
	FreshnessDesired *time.Duration `json:"freshnessDesired,omitempty"`
	IconURL          *string        `json:"iconUrl,omitempty"`
}

// TransformConfig describes a sequence of stages
type TransformConfig struct {
	Stages []*Stage `json:"stages"`
	inputs map[string]*backendInput
	stages []*backendStage
}

// Stage declares a source to operate on, and a pipeline to execute
type Stage struct {
	Label    string `json:"label,omitempty"`
	Follow   string `json:"follow,omitempty"`
	Import   string `json:"import,omitempty"`
	Pipeline string `json:"pipeline,omitempty"`
}

type backendDatasetConfig struct {
	ID               string `json:"id,omitempty"`
	Label            string `json:"label,omitempty"`
	FreshnessDesired string `json:"freshnessDesired,omitempty"`
	IconURL          string `json:"iconUrl,omitempty"`
}

type backendInput struct {
	InputName string `json:"inputName"`
	StageID   string `json:"stageID,omitempty"`
	DatasetID string `json:"datasetId,omitempty"`
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

type backendDataset struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`

	Label            string `json:"label"`
	FreshnessDesired string `json:"freshnessDesired"`
	IconURL          string `json:"iconUrl"`

	Transform struct {
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
		freshnessDesired
		iconUrl
		transform {
			current {
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

	for _, s := range b.Transform.Current.Stages {
		if err := d.Transform.addStage(&s); err != nil {
			return err
		}
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

func (t *TransformConfig) toBackend() *backendTransform {
	var outputStage string

	if len(t.Stages) > 0 {
		outputStage = t.Stages[len(t.Stages)-1].Label
	}

	return &backendTransform{
		OutputStage: outputStage,
		Stages:      t.stages,
	}
}

func (t *TransformConfig) addStage(b *backendStage) (err error) {
	defer func() {
		if err == nil {
			t.stages = append(t.stages, b)
		}
	}()

	if len(b.Input) == 0 {
		return nil
	}

	if t.inputs == nil {
		t.inputs = make(map[string]*backendInput)
	}

	s := &Stage{
		Pipeline: NewPipeline(b.Pipeline).String(),
	}

	for n, i := range b.Input {
		if i.DatasetID != "" {
			if n == 0 && i.InputName == i.DatasetID {
				// we declared an import within the stage stanza
				s.Import = i.DatasetID
			} else if t.inputs[i.InputName] == nil {
				// if name doesn't match, we must have implicitly declared an input
				t.inputs[i.InputName] = i

				inputStage := &Stage{Import: i.DatasetID}
				if i.InputName != fmt.Sprintf("stage%d", len(t.Stages)) {
					inputStage.Label = i.InputName
				}

				t.Stages = append(t.Stages, inputStage)
			}
		}
	}

	if b.StageID != fmt.Sprintf("stage%d", len(t.Stages)) {
		s.Label = b.StageID
	}

	firstInput := b.Input[0]
	var previousStage *Stage
	if n := len(t.Stages); n > 0 {
		previousStage = t.Stages[n-1]
	}

	// FML, super confusing. We only want to assign Follow if it's not the default.
	if s.Import == "" && previousStage != nil {
		var label string
		if label = previousStage.Label; label == "" {
			label = fmt.Sprintf("stage%d", len(t.Stages)-1)
		}
		if firstInput.InputName != label {
			s.Follow = firstInput.InputName
		}
	}

	t.Stages = append(t.Stages, s)
	return nil
}

func (t *TransformConfig) AddStage(s *Stage) (err error) {
	defer func() {
		if err == nil {
			t.Stages = append(t.Stages, s)
		}
	}()

	if t.inputs == nil {
		t.inputs = make(map[string]*backendInput)
	}

	// validate input
	switch {
	case len(t.Stages) == 0 && s.Import == "":
		err = fmt.Errorf("first stage must declare an import")
	case s.Import != "" && s.Follow != "":
		err = fmt.Errorf("stage has both import and follow attributes")
	case s.Pipeline == "" && s.Import == "":
		err = fmt.Errorf("stage must declare either an import or a pipeline")
	case s.Follow != "" && t.inputs[s.Follow] == nil:
		err = fmt.Errorf("stage follows undeclared stage %s", s.Follow)
	case s.Label != "" && t.inputs[s.Label] != nil:
		err = fmt.Errorf("stage %s already declared", s.Label)
	}

	if err != nil {
		return
	}

	var stageInput *backendInput

	if s.Label == "" {
		s.Label = fmt.Sprintf("stage%d", len(t.Stages))
	}

	// input only stage
	if s.Import != "" && s.Pipeline == "" {
		t.inputs[s.Label] = &backendInput{
			InputName: s.Label,
			DatasetID: s.Import,
		}
		return
	}

	if s.Import != "" {
		stageInput = &backendInput{
			InputName: s.Import, // name is mandatory for backend
			DatasetID: s.Import,
		}
	}

	if s.Follow != "" {
		stageInput = t.inputs[s.Follow]
	}

	stage := &backendStage{
		StageID:  s.Label,
		Pipeline: NewPipeline(s.Pipeline).Canonical(),
	}

	if stageInput != nil {
		stage.Input = append(stage.Input, stageInput)
	}

	for i := len(t.Stages) - 1; i >= 0; i-- {
		label := t.Stages[i].Label
		if stageInput == nil || label != stageInput.InputName {
			stage.Input = append(stage.Input, t.inputs[label])
		}
	}

	t.inputs[s.Label] = &backendInput{
		InputName: s.Label,
		StageID:   s.Label,
	}

	t.stages = append(t.stages, stage)

	return nil
}

func (c *Client) CreateDataset(workspaceID string, datasetConfig DatasetConfig, transformConfig TransformConfig) (*Dataset, error) {
	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     datasetConfig.toBackend(""),
		"transform":   transformConfig.toBackend(),
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

func decode(input interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      output,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func (c *Client) UpdateDataset(workspaceID string, ID string, datasetConfig DatasetConfig, transformConfig TransformConfig) (*Dataset, error) {

	result, err := c.Run(backendDatasetFragment+saveDatasetQuery, map[string]interface{}{
		"workspaceId": workspaceID,
		"dataset":     datasetConfig.toBackend(ID),
		"transform":   transformConfig.toBackend(),
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
