package meta

import (
	"context"
	"net/http"
	"net/url"

	"github.com/machinebox/graphql"
)

// Client implements our customer GQL API
type Client struct {
	gqlClient *graphql.Client
}

// Run raw GraphQL query against metadata endpoint
func (c *Client) Run(ctx context.Context, reqBody string, vars map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(reqBody)
	for k, v := range vars {
		req.Var(k, v)
	}

	var result map[string]interface{}
	err := c.gqlClient.Run(ctx, req, &result)
	return result, err
}

// TODO: move this to "get user id" or something
func (c *Client) Verify() error {
	if _, err := c.Run(context.Background(), `{ currentUser { id } }`, nil); err != nil {
		return err
	}
	return nil
}

// New returns client to customer API
func New(endpoint string, client *http.Client) (*Client, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	return &Client{
		gqlClient: graphql.NewClient(endpoint, graphql.WithHTTPClient(client)),
	}, nil
}
