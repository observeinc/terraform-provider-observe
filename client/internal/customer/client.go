package customer

import (
	"bytes"
	"context"
	"encoding/json"
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

// New returns client to customer API
func New(endpoint string, client *http.Client) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: client,
	}
}
