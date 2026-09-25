package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		return errors.New(strings.ToLower(http.StatusText(resp.StatusCode)))
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

const maxErrorResponseBodyBytes = 4 * 1024

func responseWrapper(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseBodyBytes))
		message := ""
		if readErr == nil {
			var errResponse errorResponse
			jsonBody := json.Unmarshal(body, &errResponse) == nil
			if jsonBody {
				message = strings.TrimSpace(errResponse.Message)
			}
			if !jsonBody {
				message = strings.TrimSpace(string(body))
			}
		}
		if message == "" {
			message = strings.ToLower(http.StatusText(resp.StatusCode))
		}
		if message == "" {
			message = "http error"
		}
		return nil, ErrorWithStatusCode{StatusCode: resp.StatusCode, Err: errors.New(message)}
	}
	return resp, nil
}

func (c *Client) request(
	method, path, contentType string,
	body io.Reader,
) (*http.Response, error) {
	return c.requestWithContext(context.Background(), method, path, contentType, body)
}

// requestWithContext returns the response for a 2xx status, which the caller
// must close; other statuses return ErrorWithStatusCode.
func (c *Client) requestWithContext(
	ctx context.Context,
	method, path, contentType string,
	body io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if err != nil {
		return nil, err
	}

	if len(contentType) > 0 {
		req.Header.Set("Content-Type", contentType)
	}

	return responseWrapper(c.httpClient.Do(req))
}

func (c *Client) Post(path, contentType string, body io.Reader) (*http.Response, error) {
	return c.request(http.MethodPost, path, contentType, body)
}

func (c *Client) Get(path string) (*http.Response, error) {
	return c.request(http.MethodGet, path, "", nil)
}

func (c *Client) Put(path string, contentType string, body io.Reader) (*http.Response, error) {
	return c.request(http.MethodPut, path, contentType, body)
}

func (c *Client) Patch(path string, contentType string, body io.Reader) (*http.Response, error) {
	return c.request(http.MethodPatch, path, contentType, body)
}

func (c *Client) Delete(path string) (*http.Response, error) {
	return c.request(http.MethodDelete, path, "", nil)
}

// New returns client to customer API
func New(endpoint string, client *http.Client) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: client,
	}
}
