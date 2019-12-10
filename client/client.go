package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/machinebox/graphql"
)

var (
	ErrUnauthorized = errors.New("authorization error")
)

type authTripper struct {
	http.RoundTripper
	key string
}

func (t *authTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// log request before adding authorization header
	if s, err := httputil.DumpRequest(req, true); err != nil {
		return nil, err
	} else {
		log.Printf("[DEBUG] %s\n", s)
	}

	if t.key != "" {
		req.Header.Set("Authorization", "Bearer "+t.key)
	}

	if t.RoundTripper == nil {
		t.RoundTripper = http.DefaultTransport
	}

	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp, err
	case http.StatusUnprocessableEntity:
		s, _ := httputil.DumpResponse(resp, true)
		log.Printf("[WARN] %s\n", s)
		return resp, err
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("received unexpected status code %d", resp.StatusCode)
	}
}

type Client struct {
	client *graphql.Client
}

// Verify checks if we can connect to API.
func (c *Client) Verify() error {
	req := graphql.NewRequest(`{ currentUser { id } }`)
	var respData struct {
		Response struct {
			Id string `json:"id"`
		} `json:"currentUser"`
	}

	if err := c.client.Run(context.Background(), req, &respData); err != nil {
		return err
	}

	return nil
}

func NewClient(baseURL string, key string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	authed := &http.Client{
		Transport: &authTripper{key: key},
	}

	c := &Client{
		client: graphql.NewClient(u.String(), graphql.WithHTTPClient(authed)),
	}

	return c, c.Verify()
}
