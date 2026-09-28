package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Dashboard models the Dashboard-Resource returned by the dashboards REST API.
// Resource reads go through GraphQL; this type decodes the create/update responses
// (chiefly for the assigned id).
type Dashboard struct {
	Id            string              `json:"id"`
	SchemaVersion int                 `json:"schemaVersion"`
	Name          string              `json:"name"`
	Definition    json.RawMessage     `json:"definition"`
	Description   string              `json:"description"`
	Visibility    string              `json:"visibility"`
	ObjectTags    map[string][]string `json:"objectTags"`
}

// DashboardCreateInput is the POST /v1/dashboards body (Dashboard-CreateRequest).
// schemaVersion, name, and definition are required; the rest are optional.
type DashboardCreateInput struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Name          string              `json:"name"`
	Definition    json.RawMessage     `json:"definition"`
	Description   *string             `json:"description,omitempty"`
	Visibility    *string             `json:"visibility,omitempty"`
	ObjectTags    map[string][]string `json:"objectTags,omitempty"`
}

// DashboardPatchInput is the PATCH /v1/dashboards/{id} body (Dashboard-UpdateRequest).
// It follows RFC 7396 merge-patch semantics: an omitted field is left unchanged, so every
// field is a pointer with omitempty and only changed fields are populated.
type DashboardPatchInput struct {
	SchemaVersion *int                 `json:"schemaVersion,omitempty"`
	Name          *string              `json:"name,omitempty"`
	Definition    json.RawMessage      `json:"definition,omitempty"`
	Description   *string              `json:"description,omitempty"`
	Visibility    *string              `json:"visibility,omitempty"`
	ObjectTags    *map[string][]string `json:"objectTags,omitempty"`
}

func (client *Client) decodeDashboardFromBody(resp *http.Response) (*Dashboard, error) {
	dashboard := &Dashboard{}
	if err := json.NewDecoder(resp.Body).Decode(dashboard); err != nil {
		return nil, err
	}
	return dashboard, nil
}

func (client *Client) CreateDashboard(ctx context.Context, input *DashboardCreateInput) (*Dashboard, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post("/v1/dashboards?expand=true", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return client.decodeDashboardFromBody(resp)
}

func (client *Client) UpdateDashboard(ctx context.Context, id string, input *DashboardPatchInput) (*Dashboard, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/dashboards/"+url.PathEscape(id)+"?expand=true", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return client.decodeDashboardFromBody(resp)
}
