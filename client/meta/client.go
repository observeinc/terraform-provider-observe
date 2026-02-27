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

	"github.com/Khan/genqlient/graphql"
)

// Client implements our customer GQL API
type Client struct {
	Gql graphql.Client
	*http.Client
	endpoint string
}

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

type graphResponse struct {
	Data   interface{}
	Errors []graphErr
}

type graphRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// Run raw GraphQL query against metadata endpoint
func (c *Client) Run(ctx context.Context, query string, vars map[string]interface{}) (map[string]interface{}, error) {
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
		// return first error
		return nil, gr.Errors[0]
	}

	return v, nil
}

// New returns client to customer API
func New(endpoint string, client *http.Client) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	gql := graphql.NewClient(endpoint, client)

	return &Client{
		endpoint: endpoint,
		Client:   client,
		Gql:      gql,
	}, nil
}
