package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"net/http"
)

// Client implements our RESTful customer API
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// do is a helper to run HTTP request for a JSON API
func (c *Client) do(ctx context.Context, method string, path string, body map[string]interface{}, result interface{}) error {
	var (
		endpoint = fmt.Sprintf("%s%s", c.endpoint, path)
		reqBody  io.Reader
	)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	default:
		return fmt.Errorf(strings.ToLower(http.StatusText(resp.StatusCode)))
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	return nil
}

type errorResponse struct {
	Message string `json:"message"`
}

func responseWrapper(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		defer resp.Body.Close()
		var errResponse errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResponse); err != nil {
			return nil, fmt.Errorf("got status code %d, but failed to decode error message: %w", resp.StatusCode, err)
		}
		return nil, ErrorWithStatusCode{StatusCode: resp.StatusCode, Err: errors.New(errResponse.Message)}
	}
	return resp, nil
}

func (c *Client) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return responseWrapper(c.httpClient.Post(c.endpoint+path, contentType, body))
}

func (c *Client) Get(path string) (*http.Response, error) {
	return responseWrapper(c.httpClient.Get(c.endpoint + path))
}

func (c *Client) Put(path string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", c.endpoint+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return responseWrapper(c.httpClient.Do(req))
}

func (c *Client) Patch(path string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", c.endpoint+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return responseWrapper(c.httpClient.Do(req))
}

func (c *Client) Delete(path string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	return responseWrapper(c.httpClient.Do(req))
}

// New returns client to customer API
func New(endpoint string, client *http.Client) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: client,
	}
}
