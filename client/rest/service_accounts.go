package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type ServiceAccountResource struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled"`
}

type ServiceAccountDefinition struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled"`
}

func (sa *ServiceAccountResource) Oid() oid.OID {
	return oid.OID{
		Id:   sa.Id,
		Type: oid.TypeUser,
	}
}

func (client *Client) decodeServiceAccountFromBody(resp *http.Response) (*ServiceAccountResource, error) {
	defer resp.Body.Close()

	resource := &ServiceAccountResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateServiceAccount(ctx context.Context, req *ServiceAccountDefinition) (*ServiceAccountResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post("/v1/service-accounts", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return client.decodeServiceAccountFromBody(resp)
}

func (client *Client) GetServiceAccount(ctx context.Context, id string) (*ServiceAccountResource, error) {
	resp, err := client.Get("/v1/service-accounts/" + id)
	if err != nil {
		return nil, err
	}
	return client.decodeServiceAccountFromBody(resp)
}

func (client *Client) UpdateServiceAccount(ctx context.Context, id string, req *ServiceAccountDefinition) (*ServiceAccountResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/service-accounts/"+id, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return client.decodeServiceAccountFromBody(resp)
}

func (client *Client) DeleteServiceAccount(ctx context.Context, id string) error {
	resp, err := client.Delete("/v1/service-accounts/" + id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (client *Client) ListServiceAccounts(ctx context.Context) ([]ServiceAccountResource, error) {
	resp, err := client.Get("/v1/service-accounts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ServiceAccounts []ServiceAccountResource `json:"serviceAccounts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.ServiceAccounts, nil
}
