package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type App struct {
	ID          string                  `json:"id"`
	WorkspaceID string                  `json:"workspace"`
	Config      *AppConfig              `json:"config"`
	Status      *AppStatus              `json:"status"`
	Outputs     *map[string]interface{} `json:"outputs"`
}

type AppConfig struct {
	ModuleId  string            `json:"moduleId"`
	Version   string            `json:"version"`
	Folder    *OID              `json:"folder"`
	Variables map[string]string `json:"variables"`
}

type AppStatus struct {
	State string `json:"state"`
}

func (a *App) OID() *OID {
	return &OID{
		Type: TypeApp,
		ID:   a.ID,
	}
}

func (config *AppConfig) toGQL() (*meta.AppInput, error) {
	appInput := &meta.AppInput{
		ModuleId: config.ModuleId,
		Version:  config.Version,
		FolderID: toObjectPointer(config.Folder.Version),
	}

	for key, value := range config.Variables {
		appInput.Variables = append(appInput.Variables, meta.AppVariableInput{
			Name:  key,
			Value: value,
		})
	}

	return appInput, nil
}

func newApp(c *meta.App) (*App, error) {

	folderId := c.FolderID.String()

	config := &AppConfig{
		ModuleId: c.Config.ModuleId,
		Version:  c.Config.Version,
		Folder: &OID{
			Type:    TypeFolder,
			ID:      c.WorkspaceID.String(),
			Version: &folderId,
		},
	}

	for _, el := range c.Config.Variables {
		if el.Value != nil {
			config.Variables[el.Name] = *el.Value
		}
	}

	return &App{
		ID:          c.ID.String(),
		WorkspaceID: c.WorkspaceID.String(),
		Config:      config,
		Status: &AppStatus{
			State: c.Status.State,
		},
		Outputs: c.Outputs,
	}, nil
}
