package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type Client struct {
	endpoint string
	*http.Client
}

// New returns client to REST API
func New(endpoint string, client *http.Client) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	u.Path = path.Join(u.Path, "/v1")

	return &Client{
		endpoint: u.String(),
		Client:   client,
	}, nil
}
