package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

// Dataset is the output of a sequence of stages operating on a collection of inputs
type Datastream struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	Config      *DatastreamConfig `json:"config"`
}

// DatastreamConfig contains configurable elements associated to Datastream
type DatastreamConfig struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IconURL     *string `json:"icon_url"`
}

func (d *Datastream) OID() *OID {
	return &OID{
		Type: TypeDatastream,
		ID:   d.ID,
	}
}

func newDatastream(gqlDatastream *meta.Datastream) (d *Datastream, err error) {
	d = &Datastream{
		ID:          gqlDatastream.ID.String(),
		WorkspaceID: gqlDatastream.WorkspaceId.String(),
		Config: &DatastreamConfig{
			Name:        gqlDatastream.Name,
			Description: gqlDatastream.Description,
			IconURL:     gqlDatastream.IconURL,
		},
	}
	return
}

func (c *DatastreamConfig) Validate() error {
	_, err := c.toGQL()
	return err
}

func (c *DatastreamConfig) toGQL() (*meta.DatastreamInput, error) {
	if c.Name == "" {
		return nil, errNameMissing
	}

	datastreamInput := &meta.DatastreamInput{
		Name:        c.Name,
		Description: c.Description,
		IconURL:     c.IconURL,
	}

	return datastreamInput, nil
}
