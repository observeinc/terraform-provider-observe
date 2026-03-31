package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Client implements our customer GQL API
type Client struct {
	Gql graphql.Client
	*http.Client
	endpoint    string
	retryConfig RetryConfig
}

type graphResponse struct {
	Data   interface{}
	Errors gqlerror.List
}

// Run raw GraphQL query against metadata endpoint
func (c *Client) Run(ctx context.Context, query string, vars map[string]interface{}) (map[string]interface{}, error) {
	var v map[string]interface{}
	err := withRetry(ctx, c.retryConfig, func() error {
		var runErr error
		v, runErr = c.doRun(ctx, query, vars)
		return runErr
	})
	return v, err
}

func (c *Client) doRun(ctx context.Context, query string, vars map[string]interface{}) (map[string]interface{}, error) {
	var requestBody bytes.Buffer

	err := json.NewEncoder(&requestBody).Encode(map[string]interface{}{
		"query":     query,
		"variables": vars,
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding body: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json; charset=utf-8")

	res, err := c.Do(r)
	if err != nil {
		return nil, fmt.Errorf("error processing request: %w", err)
	}
	defer res.Body.Close()

	if code := res.StatusCode; code != http.StatusOK {
		return nil, errors.New(strings.ToLower(http.StatusText(code)))
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	var v map[string]interface{}
	gr := &graphResponse{
		Data: &v,
	}

	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(gr.Errors) > 0 {
		return nil, gr.Errors
	}

	return v, nil
}

type Option func(*Client)

// WithRetries enables automatic retries for retryable GraphQL errors (e.g.
// RESOURCE_EXHAUSTED) with exponential backoff starting at initialBackoff.
// Both maxRetries and initialBackoff must be positive for retries to be enabled.
func WithRetries(maxRetries int, initialBackoff time.Duration) Option {
	return func(c *Client) {
		if maxRetries > 0 && initialBackoff > 0 {
			c.retryConfig = RetryConfig{
				MaxRetries:     maxRetries,
				InitialBackoff: initialBackoff,
			}
		}
	}
}

// New returns client to customer API
func New(endpoint string, httpClient *http.Client, opts ...Option) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	c := &Client{
		endpoint: endpoint,
		Client:   httpClient,
	}
	for _, opt := range opts {
		opt(c)
	}

	c.Gql = newRetryClient(graphql.NewClient(endpoint, httpClient), c.retryConfig)

	return c, nil
}
