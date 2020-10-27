package api

import (
	"context"
	"net/http"

	"github.com/machinebox/graphql"
)

// Client implements our current API given an interface that can speak GraphQL
type Client struct {
	gqlClient *graphql.Client
}

// Run raw GraphQL query against API
func (c *Client) Run(reqBody string, vars map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(reqBody)
	for k, v := range vars {
		req.Var(k, v)
	}

	var result map[string]interface{}
	err := c.gqlClient.Run(context.Background(), req, &result)
	return result, err
}

// New returns client to meta API
func New(endpoint string, client *http.Client) *Client {
	return &Client{
		gqlClient: graphql.NewClient(endpoint, graphql.WithHTTPClient(client)),
	}
}
