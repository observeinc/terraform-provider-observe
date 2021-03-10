package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Channel struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace"`
	Config      *ChannelConfig `json:"config"`
}

type ChannelConfig struct {
	Name        string  `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`
	Monitors    []*OID  `json:"monitors"`
	Actions     []*OID  `json:"actions"`
}

func (c *Channel) OID() *OID {
	return &OID{
		Type: TypeChannel,
		ID:   c.ID,
	}
}

func (config *ChannelConfig) toGQL() (*meta.ChannelInput, []string, []string, error) {
	channelInput := &meta.ChannelInput{
		Name:        config.Name,
		IconURL:     config.IconURL,
		Description: config.Description,
	}

	// need to convert from OID to regular ID

	// actions must never be nil, otherwise we will never clear value in the
	// case where we no longer are subscribed to any
	actions := make([]string, len(config.Actions))
	for i, v := range config.Actions {
		actions[i] = v.ID
	}

	monitors := make([]string, len(config.Monitors))
	for i, v := range config.Monitors {
		monitors[i] = v.ID
	}

	return channelInput, actions, monitors, nil
}

func newChannel(c *meta.Channel) (*Channel, error) {
	config := &ChannelConfig{
		Name:        c.Name,
		IconURL:     c.IconURL,
		Description: c.Description,
	}

	for _, channelAction := range c.Actions {
		oid := &OID{Type: TypeChannelAction, ID: channelAction.ID.String()}
		config.Actions = append(config.Actions, oid)
	}

	for _, monitor := range c.Monitors {
		oid := &OID{Type: TypeMonitor, ID: monitor.ID.String()}
		config.Monitors = append(config.Monitors, oid)
	}

	return &Channel{
		ID:          c.ID.String(),
		WorkspaceID: c.WorkspaceId.String(),
		Config:      config,
	}, nil
}
