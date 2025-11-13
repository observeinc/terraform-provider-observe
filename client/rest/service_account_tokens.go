package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ServiceAccountTokenResource models ApiToken-Resource from OpenAPI
// Note: secret is only returned on create.
type ServiceAccountTokenResource struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Expiration  string `json:"expiration"`
	Disabled    bool   `json:"disabled"`
	Secret      *string `json:"secret,omitempty"`
}

type ServiceAccountTokenCreateRequest struct {
	Label         string  `json:"label"`
	Description   string  `json:"description,omitempty"`
	LifetimeHours int     `json:"lifetimeHours"`
}

type ServiceAccountTokenUpdateRequest struct {
	Label         *string `json:"label,omitempty"`
	Description   *string `json:"description,omitempty"`
	LifetimeHours *int    `json:"lifetimeHours,omitempty"`
	Disabled      *bool   `json:"disabled,omitempty"`
}

func (client *Client) decodeServiceAccountTokenFromBody(resp *http.Response) (*ServiceAccountTokenResource, error) {
	defer resp.Body.Close()
	resource := &ServiceAccountTokenResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateServiceAccountToken(ctx context.Context, accountId string, req *ServiceAccountTokenCreateRequest) (*ServiceAccountTokenResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post("/v1/service-accounts/"+url.PathEscape(accountId)+"/api-tokens", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeServiceAccountTokenFromBody(resp)
}

func (client *Client) GetServiceAccountToken(ctx context.Context, accountId, tokenId string) (*ServiceAccountTokenResource, error) {
	resp, err := client.Get("/v1/service-accounts/" + url.PathEscape(accountId) + "/api-tokens/" + url.PathEscape(tokenId))
	if err != nil {
		return nil, err
	}
	return client.decodeServiceAccountTokenFromBody(resp)
}

func (client *Client) UpdateServiceAccountToken(ctx context.Context, accountId, tokenId string, req *ServiceAccountTokenUpdateRequest) (*ServiceAccountTokenResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/service-accounts/"+url.PathEscape(accountId)+"/api-tokens/"+url.PathEscape(tokenId), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeServiceAccountTokenFromBody(resp)
}

func (client *Client) DeleteServiceAccountToken(ctx context.Context, accountId, tokenId string) error {
	resp, err := client.Delete("/v1/service-accounts/" + url.PathEscape(accountId) + "/api-tokens/" + url.PathEscape(tokenId))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

