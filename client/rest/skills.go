package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// SkillVisibility is the internal REST API visibility (OpenAPI Skill-Visibility enum).
type SkillVisibility string

const (
	SkillVisibilityListed   SkillVisibility = "Listed"
	SkillVisibilityUnlisted SkillVisibility = "Unlisted"
)

type SkillResource struct {
	Id          string          `json:"id"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Content     string          `json:"content"`
	Visibility  SkillVisibility `json:"visibility"`
	CreatedBy   SkillUser       `json:"createdBy"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedBy   SkillUser       `json:"updatedBy"`
	UpdatedAt   string          `json:"updatedAt"`
}

type SkillUser struct {
	Id    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// All fields are pointers with omitempty to support PATCH semantics.
type SkillUpdateRequest struct {
	Label       *string          `json:"label,omitempty"`
	Description *string          `json:"description,omitempty"`
	Content     *string          `json:"content,omitempty"`
	Visibility  *SkillVisibility `json:"visibility,omitempty"`
}

type SkillCreateRequest struct {
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Content     string          `json:"content"`
	Visibility  SkillVisibility `json:"visibility"`
}

func (r *SkillResource) Oid() oid.OID {
	return oid.OID{
		Id:   r.Id,
		Type: oid.TypeSkill,
	}
}

func (client *Client) decodeSkillResourceFromBody(resp *http.Response) (*SkillResource, error) {
	defer resp.Body.Close()
	resource := &SkillResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateSkill(ctx context.Context, req *SkillCreateRequest) (*SkillResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post("/v1/skills?expand=true", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeSkillResourceFromBody(resp)
}

func (client *Client) GetSkill(ctx context.Context, id string) (*SkillResource, error) {
	resp, err := client.Get("/v1/skills/" + url.PathEscape(id) + "?expand=true")
	if err != nil {
		return nil, err
	}
	return client.decodeSkillResourceFromBody(resp)
}

func (client *Client) UpdateSkill(ctx context.Context, id string, req *SkillUpdateRequest) (*SkillResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/skills/"+url.PathEscape(id)+"?expand=true", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeSkillResourceFromBody(resp)
}

func (client *Client) DeleteSkill(ctx context.Context, id string) error {
	resp, err := client.Delete("/v1/skills/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
