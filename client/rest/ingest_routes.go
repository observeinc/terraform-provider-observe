package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// IngestRouteResource is an ingest route returned by the REST API.
type IngestRouteResource struct {
	Id                     string                 `json:"id"`
	Type                   string                 `json:"type"`
	Pipeline               string                 `json:"pipeline"`
	Layout                 map[string]interface{} `json:"layout"`
	DestinationId          string                 `json:"destinationId"`
	SecondaryDestinationId *string                `json:"secondaryDestinationId"`
	Enabled                bool                   `json:"enabled"`
	ManagedBy              *IngestRouteObjectRef  `json:"managedBy"`
}

// IngestRouteObjectRef identifies an Observe object without expanding its record.
type IngestRouteObjectRef struct {
	Id string `json:"id"`
}

// IngestRouteCreateRequest creates a non-default route.
type IngestRouteCreateRequest struct {
	Pipeline               string  `json:"pipeline"`
	DestinationId          string  `json:"destinationId"`
	SecondaryDestinationId *string `json:"secondaryDestinationId,omitempty"`
}

// IngestRouteUpdateRequest is the route merge-patch payload. A nil secondary
// destination is serialized as JSON null and clears the secondary destination.
type IngestRouteUpdateRequest struct {
	Pipeline                        *string `json:"pipeline,omitempty"`
	DestinationId                   *string `json:"destinationId,omitempty"`
	SecondaryDestinationId          *string `json:"-"`
	SecondaryDestinationIdSpecified bool    `json:"-"`
	Enabled                         *bool   `json:"enabled,omitempty"`
}

// IngestRoutePriorityUpdateRequest replaces the order of routes for one type.
type IngestRoutePriorityUpdateRequest struct {
	Ordering []string `json:"ordering"`
}

// MarshalJSON overrides the struct tags to serialize a specified nil secondary
// destination as JSON null.
func (request IngestRouteUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]interface{}{}
	if request.Pipeline != nil {
		body["pipeline"] = *request.Pipeline
	}
	if request.DestinationId != nil {
		body["destinationId"] = *request.DestinationId
	}
	if request.SecondaryDestinationIdSpecified {
		body["secondaryDestinationId"] = request.SecondaryDestinationId
	}
	if request.Enabled != nil {
		body["enabled"] = *request.Enabled
	}
	return json.Marshal(body)
}

func (client *Client) decodeIngestRouteFromBody(resp *http.Response) (*IngestRouteResource, error) {
	defer resp.Body.Close()
	route := &IngestRouteResource{}
	if err := json.NewDecoder(resp.Body).Decode(route); err != nil {
		return nil, err
	}
	return route, nil
}

func (client *Client) CreateIngestRoute(ctx context.Context, routeType string, req *IngestRouteCreateRequest) (*IngestRouteResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Post(ingestRoutePath(routeType), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestRouteFromBody(resp)
}

func (client *Client) GetIngestRoute(ctx context.Context, routeType, id string) (*IngestRouteResource, error) {
	resp, err := client.Get(ingestRoutePath(routeType) + "/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestRouteFromBody(resp)
}

func (client *Client) ListIngestRoutes(ctx context.Context, routeType string) ([]IngestRouteResource, error) {
	resp, err := client.Get(ingestRoutePath(routeType))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		IngestRoutes []IngestRouteResource `json:"ingestRoutes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.IngestRoutes, nil
}

func (client *Client) UpdateIngestRoute(ctx context.Context, routeType, id string, req *IngestRouteUpdateRequest) (*IngestRouteResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch(ingestRoutePath(routeType)+"/"+url.PathEscape(id), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return client.decodeIngestRouteFromBody(resp)
}

func (client *Client) UpdateIngestRoutePriorities(ctx context.Context, routeType string, ordering []string) ([]IngestRouteResource, error) {
	body, err := json.Marshal(IngestRoutePriorityUpdateRequest{Ordering: ordering})
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch(ingestRoutePath(routeType), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		IngestRoutes []IngestRouteResource `json:"ingestRoutes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.IngestRoutes, nil
}

func (client *Client) DeleteIngestRoute(ctx context.Context, routeType, id string) error {
	resp, err := client.Delete(ingestRoutePath(routeType) + "/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func ingestRoutePath(routeType string) string {
	return "/v1/ingest/routes/" + url.PathEscape(routeType)
}
