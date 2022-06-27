package client

import (
	"encoding/json"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Dashboard struct {
	ID        string           `json:"id"`
	Workspace *OID             `json:"workspace"`
	Config    *DashboardConfig `json:"config"`
}

func (b *Dashboard) OID() *OID {
	return &OID{
		Type: TypeDashboard,
		ID:   b.ID,
	}
}

type DashboardConfig struct {
	Name            string  `json:"name"`
	Icon            *string `json:"iconUrl"`
	Stages          *string `json:"stages"`
	Layout          *string `json:"layout"`
	Parameters      *string `json:"parameters"`
	ParameterValues *string `json:"parameterValues"`
}

func (dc *DashboardConfig) toGQL() (*meta.DashboardInput, error) {
	d := &meta.DashboardInput{
		Name: dc.Name,
		Icon: dc.Icon,
	}

	if err := maybeUnmarshal(dc.Stages, &d.Stages, "dashboard stages"); err != nil {
		return nil, err
	}
	if err := maybeUnmarshal(dc.Layout, &d.Layout, "dashboard layout"); err != nil {
		return nil, err
	}
	if err := maybeUnmarshal(dc.Parameters, &d.Parameters, "dashboard parameters"); err != nil {
		return nil, err
	}
	if err := maybeUnmarshal(dc.ParameterValues, &d.ParameterValues, "dashboard parameter values"); err != nil {
		return nil, err
	}

	return d, nil
}

func maybeUnmarshal(rawJson *string, dst interface{}, name string) error {
	if rawJson == nil {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(*rawJson), &v); err != nil {
		return fmt.Errorf("failed to unmarshal %s as json: %w", name, err)
	}
	decoder, err := meta.NewDecoder(true, dst)
	if err != nil {
		return fmt.Errorf("failed to create decoder for %s: %w", name, err)
	}
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to map JSON to %s: %w", name, err)
	}
	return nil
}

func newDashboard(d *meta.Dashboard) (*Dashboard, error) {
	dc := &DashboardConfig{
		Name: d.Name,
		Icon: d.IconUrl,
	}

	if err := marshal(d.Stages, &dc.Stages, "dashboard stages"); err != nil {
		return nil, err
	}
	if d.Layout != nil {
		if err := marshal(d.Layout, &dc.Layout, "dashboard layout"); err != nil {
			return nil, err
		}
	}
	if d.Parameters != nil {
		if err := marshal(d.Parameters, &dc.Parameters, "dashboard parameters"); err != nil {
			return nil, err
		}
	}
	if d.ParameterValues != nil {
		if err := marshal(d.ParameterValues, &dc.ParameterValues, "dashboard parameter values"); err != nil {
			return nil, err
		}
	}

	return &Dashboard{
		ID: d.ID.String(),
		Workspace: &OID{
			Type: TypeWorkspace,
			ID:   d.WorkspaceId.String(),
		},
		Config: dc,
	}, nil
}

func marshal(src interface{}, dst **string, name string) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", name, err)
	}
	s := string(data)
	*dst = &s
	return nil
}
