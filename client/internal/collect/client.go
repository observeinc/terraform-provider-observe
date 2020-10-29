package collect

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
)

// Client implements our current API given an interface that can speak GraphQL
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// New returns client to collect API
func New(endpoint string, client *http.Client) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	u.Path = path.Join(u.Path, "/v1/observations")

	return &Client{
		endpoint:   u.String(),
		httpClient: client,
	}, nil
}
