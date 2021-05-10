package client

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
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
	*Query
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	IconURL     *string        `json:"icon_url"`
	Freshness   *time.Duration `json:"freshness"`

	// in practice PathCost is mandatory, since it cannot be set to null
	PathCost int64 `json:"path_cost"`
}

func (d *Dataset) OID() *OID {
	return &OID{
		Type:    TypeDataset,
		ID:      d.ID,
		Version: &d.Version,
	}
}

func newDataset(gqlDataset *meta.Dataset) (d *Dataset, err error) {

	var pathCost int64
	if gqlDataset.PathCost != nil {
		pathCost = *gqlDataset.PathCost
	}

	d = &Dataset{
		ID:          gqlDataset.ID.String(),
		WorkspaceID: gqlDataset.WorkspaceId.String(),
		Version:     gqlDataset.LastSaved,
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
		if targetDataset := gqlFk.TargetDataset; targetDataset != nil {
			id := meta.ObjectIdScalarPointer(*targetDataset).String()
			fk := ForeignKeyConfig{
				Label:     gqlFk.Label,
				Target:    &id,
				SrcFields: gqlFk.SrcFields,
				DstFields: gqlFk.DstFields,
			}
			d.ForeignKeys = append(d.ForeignKeys, fk)
		}
	}

	if gqlDataset.Transform.Current == nil {
		// Observation table has no transform, is still valid
		return
	}

	d.Config.Query, err = newQuery(gqlDataset.Transform.Current.Query)
	if err != nil {
		return nil, err
	}
	return
}

// Validate verifies dataset config
func (c *DatasetConfig) Validate() error {
	_, _, err := c.toGQL()
	return err
}

func (c *DatasetConfig) toGQLDatasetInput() (*meta.DatasetInput, error) {
	if c.Name == "" {
		return nil, errNameMissing
	}

	datasetInput := &meta.DatasetInput{
		Label:           c.Name,
		Description:     c.Description,
		IconURL:         c.IconURL,
		OverwriteSource: true,
	}

	i := fmt.Sprintf("%d", c.PathCost)
	datasetInput.PathCost = &i

	if c.Freshness != nil {
		i := fmt.Sprintf("%d", c.Freshness.Nanoseconds())
		datasetInput.FreshnessDesired = &i
	}
	return datasetInput, nil
}

func (c *DatasetConfig) toGQL() (*meta.DatasetInput, *meta.MultiStageQueryInput, error) {
	datasetInput, err := c.toGQLDatasetInput()
	if err != nil {
		return nil, nil, err
	}

	queryInput, err := c.Query.toGQL()
	if err != nil {
		return nil, nil, err
	}
	return datasetInput, queryInput, nil
}

func invalidObjectID(s *string) bool {
	if s == nil {
		return false
	}
	_, err := strconv.ParseInt(*s, 10, 64)
	return err != nil
}

func toObjectPointer(s *string) *meta.ObjectIdScalar {
	if s == nil {
		return nil
	}
	i, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		panic(err)
	}
	return meta.ObjectIdScalarPointer(i)
}
