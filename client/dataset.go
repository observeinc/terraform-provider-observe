package client

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/api"
)

var (
	errObjectIDInvalid      = errors.New("object id is invalid")
	errNameMissing          = errors.New("name not set")
	errInputsMissing        = errors.New("no inputs defined")
	errStagesMissing        = errors.New("no stages defined")
	errInputNameMissing     = errors.New("name not set")
	errInputEmpty           = errors.New("dataset not set")
	errNameConflict         = errors.New("name already declared")
	errStageInputUnresolved = errors.New("input could not be resolved")
	errStageInputMissing    = errors.New("input missing")
	errStagePipelineMissing = errors.New("pipeline not set")
)

// Dataset is the output of a sequence of stages operating on a collection of inputs
type Dataset struct {
	ID          string             `json:"id"`
	WorkspaceID string             `json:"workspace_id"`
	Version     string             `json:"version"`
	Config      *DatasetConfig     `json:"config"`
	ForeignKeys []ForeignKeyConfig `json:"foreign_keys"`
}

// DatasetConfig contains configurable elements associated to Dataset
type DatasetConfig struct {
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	IconURL     *string           `json:"icon_url"`
	Freshness   *time.Duration    `json:"freshness"`
	Inputs      map[string]*Input `json:"inputs"`
	Stages      []*Stage          `json:"stages"`

	// in practice PathCost is mandatory, since it cannot be set to null
	PathCost int64 `json:"path_cost"`
}

// Stage applies a pipeline to an input
// If no input is provided, stage will follow on from previous stage
// An alias must be provided for callers to be able to reference this stage in OPAL
// Internally, the alias does not map to the stageID - it is the input name we
// use when refering to this stage
type Stage struct {
	Alias    *string `json:"alias,omitempty"`
	Input    *string `json:"input,omitempty"`
	Pipeline string  `json:"pipeline"`
}

// Input references an existing data source
type Input struct {
	Dataset *string ` json:"dataset,omitempty"`
}

func (d *Dataset) OID() *OID {
	return &OID{
		Type:    TypeDataset,
		ID:      d.ID,
		Version: &d.Version,
	}
}

func newDataset(gqlDataset *api.Dataset) (*Dataset, error) {

	var pathCost int64
	if gqlDataset.PathCost != nil {
		pathCost = *gqlDataset.PathCost
	}

	d := &Dataset{
		ID:          gqlDataset.ID.String(),
		WorkspaceID: gqlDataset.WorkspaceId.String(),
		Version:     gqlDataset.Version,
		Config: &DatasetConfig{
			Name:        gqlDataset.Label,
			Description: gqlDataset.Description,
			IconURL:     gqlDataset.IconURL,
			Freshness:   gqlDataset.FreshnessDesired,
			PathCost:    pathCost,
		},
	}

	// foreignKeys attribute is read only, but we'll re-use the
	// ForeignKeyConfig struct to populate the list
	for _, gqlFk := range gqlDataset.ForeignKeys {
		id := api.ObjectIdScalarPointer(*gqlFk.TargetDataset).String()
		fk := ForeignKeyConfig{
			Label:     gqlFk.Label,
			Target:    &id,
			SrcFields: gqlFk.SrcFields,
			DstFields: gqlFk.DstFields,
		}
		d.ForeignKeys = append(d.ForeignKeys, fk)
	}

	if gqlDataset.Transform.Current == nil {
		// Observation table has no transform, is still valid
		return d, nil
	}

	// first reconstruct all inputs
	stageIDs := make(map[string]string)
	d.Config.Inputs = make(map[string]*Input)
	for _, stageQuery := range gqlDataset.Transform.Current.Stages {
		for _, i := range stageQuery.Input {
			if i.DatasetID != nil {
				datasetID := i.DatasetID.String()
				d.Config.Inputs[i.InputName] = &Input{Dataset: &datasetID}
			}
			if i.StageID != "" {
				stageIDs[i.StageID] = i.InputName
			}
		}
	}

	for i, gqlStage := range gqlDataset.Transform.Current.Stages {
		stage := &Stage{
			Pipeline: gqlStage.Pipeline,
		}

		if name, ok := stageIDs[gqlStage.StageID]; ok && name != gqlStage.StageID {
			stage.Alias = &name
		}

		inputName := gqlStage.Input[0].InputName

		switch {
		case i == 0 && len(d.Config.Inputs) == 1:
			// defaulted to first input
		case i > 0 && d.Config.Stages[i-1].Alias != nil && inputName == *(d.Config.Stages[i-1].Alias):
			// follow on from aliased stage
		case stageIDs[inputName] != "":
			// follow on from anonymous stage
		default:
			stage.Input = &inputName
		}

		d.Config.Stages = append(d.Config.Stages, stage)
	}

	return d, nil
}

// Validate verifies dataset config
func (c *DatasetConfig) Validate() error {
	_, _, err := c.toGQL()
	return err
}

func (c *DatasetConfig) validateInput(i *Input) error {
	switch {
	case invalidObjectID(i.Dataset):
		return fmt.Errorf("dataset: %w", errObjectIDInvalid)
	case i.Dataset == nil:
		return errInputEmpty
	}
	return nil
}

func (c *DatasetConfig) toGQLDatasetInput() (*api.DatasetInput, error) {
	if c.Name == "" {
		return nil, errNameMissing
	}

	datasetInput := &api.DatasetInput{
		Label:       c.Name,
		Description: c.Description,
		IconURL:     c.IconURL,
	}

	i := fmt.Sprintf("%d", c.PathCost)
	datasetInput.PathCost = &i

	if c.Freshness != nil {
		i := fmt.Sprintf("%d", c.Freshness.Nanoseconds())
		datasetInput.FreshnessDesired = &i
	}
	return datasetInput, nil
}

func (c *DatasetConfig) toGQLTransformInput() (*api.TransformInput, error) {
	var transformInput api.TransformInput

	// validate and convert all inputs
	var sortedNames []string
	gqlInputs := make(map[string]*api.InputDefinitionInput, len(c.Inputs))
	for name, input := range c.Inputs {
		if err := c.validateInput(input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		gqlInputs[name] = &api.InputDefinitionInput{
			InputName: name,
			DatasetID: toObjectPointer(input.Dataset),
		}
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var defaultInput *api.InputDefinitionInput
	switch len(c.Inputs) {
	case 0:
		return nil, errInputsMissing
	case 1:
		// in only one input is provided, use it as input for first stage
		defaultInput = gqlInputs[sortedNames[0]]
	}

	// We're now ready to convert stages
	// If a stage is named, it can be used as an input for every subsequent stage.
	// If a stage is anonymous, it can still be used as a default input on the next stage.
	for i, stage := range c.Stages {
		if stage.Pipeline == "" {
			return nil, fmt.Errorf("stage %d: %w", i, errStagePipelineMissing)
		}

		// Each stage will be given an ID based on the hash of all preceeding pipelines
		gqlStage := &api.StageQueryInput{
			StageID:  fmt.Sprintf("stage-%d", i),
			Pipeline: stage.Pipeline,
		}

		// if stage has a declared input, update defaultInput
		if stage.Input != nil {
			v, ok := gqlInputs[*stage.Input]
			if !ok {
				return nil, fmt.Errorf("stage-%d: %q: %w", i, *stage.Input, errStageInputUnresolved)
			}
			defaultInput = v
		}

		if defaultInput == nil {
			return nil, fmt.Errorf("stage-%d: %w", i, errStageInputMissing)
		}

		// construct stage inputs, first default, then any declared input that
		// is referenced in pipeline.
		gqlStage.Input = append(gqlStage.Input, *defaultInput)

		for _, name := range sortedNames {
			gqlInput := gqlInputs[name]
			// don't add defaultInput a second time
			if gqlInput != defaultInput && strings.Contains(stage.Pipeline, "@"+gqlInput.InputName) {
				gqlStage.Input = append(gqlStage.Input, *gqlInput)
			}
		}

		// stage is done, append to transform
		transformInput.Stages = append(transformInput.Stages, gqlStage)
		transformInput.OutputStage = gqlStage.StageID

		// prepare for next iteration of loop
		// this stage will become defaultInput for the next
		defaultInput = &api.InputDefinitionInput{
			InputName: gqlStage.StageID,
			StageID:   gqlStage.StageID,
		}

		// if explicitly named, this stage can be also be an input for the next
		if stage.Alias != nil {
			defaultInput.InputName = *stage.Alias
			// conflict?
			gqlInputs[*stage.Alias] = defaultInput
			sortedNames = append(sortedNames, *stage.Alias)
		}
	}

	// a transform must have at least one stage
	if transformInput.OutputStage == "" {
		return nil, errStagesMissing
	}

	return &transformInput, nil
}

func (c *DatasetConfig) toGQL() (*api.DatasetInput, *api.TransformInput, error) {
	datasetInput, err := c.toGQLDatasetInput()
	if err != nil {
		return nil, nil, err
	}

	transformInput, err := c.toGQLTransformInput()
	if err != nil {
		return nil, nil, err
	}
	return datasetInput, transformInput, nil
}

func invalidObjectID(s *string) bool {
	if s == nil {
		return false
	}
	_, err := strconv.ParseInt(*s, 10, 64)
	return err != nil
}

func toObjectPointer(s *string) *api.ObjectIdScalar {
	if s == nil {
		return nil
	}
	i, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		panic(err)
	}
	return api.ObjectIdScalarPointer(i)
}
