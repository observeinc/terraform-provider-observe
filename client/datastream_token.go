package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type DatastreamToken struct {
	ID           string                 `json:"id"`
	DatastreamID string                 `json:"datastream_id"`
	Config       *DatastreamTokenConfig `json:"config"`
}

// DatastreamTokenConfig contains configurable elements associated to DatastreamToken
type DatastreamTokenConfig struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Disabled    *bool   `json:"disabled"`
	Secret      *string `json:"secret"`
}

func (d *DatastreamToken) OID() *OID {
	return &OID{
		Type: TypeDatastreamToken,
		ID:   d.ID,
	}
}

func newDatastreamToken(gql *meta.DatastreamToken) (d *DatastreamToken, err error) {
	d = &DatastreamToken{
		ID:           gql.ID,
		DatastreamID: gql.DatastreamID.String(),
		Config: &DatastreamTokenConfig{
			Name:        gql.Name,
			Description: gql.Description,
			Disabled:    &gql.Disabled,
			Secret:      gql.Secret,
		},
	}
	return
}

func (c *DatastreamTokenConfig) toGQL() (*meta.DatastreamTokenInput, error) {
	if c.Name == "" {
		return nil, errNameMissing
	}

	datastreamTokenInput := &meta.DatastreamTokenInput{
		Name:        c.Name,
		Description: c.Description,
		Disabled:    c.Disabled,
	}

	return datastreamTokenInput, nil
}
