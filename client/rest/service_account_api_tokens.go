package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ServiceAccountApiTokenResource struct {
	Id          string    `json:"id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Expiration  time.Time `json:"expiration"`
	CreatedBy   User      `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedBy   User      `json:"updatedBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Disabled    bool      `json:"disabled"`
	Secret      *string   `json:"secret,omitempty"` // Only returned on create
}

type ServiceAccountApiTokenCreateRequest struct {
	Label         string `json:"label"`
	Description   string `json:"description,omitempty"`
	LifetimeHours int    `json:"lifetimeHours"`
}

type ServiceAccountApiTokenUpdateRequest struct {
	Label         *string `json:"label,omitempty"`
	Description   *string `json:"description,omitempty"`
	LifetimeHours *int    `json:"lifetimeHours,omitempty"`
	Disabled      *bool   `json:"disabled,omitempty"`
}

type User struct {
	Id    string `json:"id"`
	Label string `json:"label"`
}

func (client *Client) decodeServiceAccountApiTokenFromBody(resp *http.Response) (*ServiceAccountApiTokenResource, error) {
	defer resp.Body.Close()

	resource := &ServiceAccountApiTokenResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateServiceAccountApiToken(ctx context.Context, accountId string, req *ServiceAccountApiTokenCreateRequest) (*ServiceAccountApiTokenResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post(
		fmt.Sprintf("/v1/service-accounts/%s/api-tokens", url.PathEscape(accountId)),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	return client.decodeServiceAccountApiTokenFromBody(resp)
}

func (client *Client) GetServiceAccountApiToken(ctx context.Context, accountId, tokenId string) (*ServiceAccountApiTokenResource, error) {
	resp, err := client.Get(
		fmt.Sprintf("/v1/service-accounts/%s/api-tokens/%s",
			url.PathEscape(accountId),
			url.PathEscape(tokenId),
		),
	)
	if err != nil {
		return nil, err
	}
	return client.decodeServiceAccountApiTokenFromBody(resp)
}

func (client *Client) UpdateServiceAccountApiToken(ctx context.Context, accountId, tokenId string, req *ServiceAccountApiTokenUpdateRequest) (*ServiceAccountApiTokenResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch(
		fmt.Sprintf("/v1/service-accounts/%s/api-tokens/%s",
			url.PathEscape(accountId),
			url.PathEscape(tokenId),
		),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	return client.decodeServiceAccountApiTokenFromBody(resp)
}

func (client *Client) DeleteServiceAccountApiToken(ctx context.Context, accountId, tokenId string) error {
	resp, err := client.Delete(
		fmt.Sprintf("/v1/service-accounts/%s/api-tokens/%s",
			url.PathEscape(accountId),
			url.PathEscape(tokenId),
		),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (client *Client) ListServiceAccountApiTokens(ctx context.Context, accountId string) ([]ServiceAccountApiTokenResource, error) {
	resp, err := client.Get(
		fmt.Sprintf("/v1/service-accounts/%s/api-tokens", url.PathEscape(accountId)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ApiTokens []ServiceAccountApiTokenResource `json:"apiTokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.ApiTokens, nil
}

