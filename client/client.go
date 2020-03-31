package client

import (
	"context"
	"crypto/tls"
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

type ResultStatus struct {
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"errorMessage"`
	DetailedInfo map[string]interface{} `json:"detailedInfo"`
}

func (s *ResultStatus) Error() error {
	if s.ErrorMessage != "" {
		return fmt.Errorf("request failed: %q", s.ErrorMessage)
	} else if !s.Success {
		return errors.New("request failed")
	}
	return nil
}

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

	s, _ := httputil.DumpResponse(resp, true)
	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("[DEBUG] %s\n", s)
		return resp, err
	case http.StatusUnprocessableEntity:
		log.Printf("[WARN] %s\n", s)
		return resp, err
	case http.StatusUnauthorized:
		log.Printf("[WARN] %s\n", s)
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

// Run raw GraphQL query against API
func (c *Client) Run(reqBody string, vars map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(reqBody)
	for k, v := range vars {
		req.Var(k, v)
	}

	var result map[string]interface{}
	err := c.client.Run(context.Background(), req, &result)
	return result, err
}

func NewClient(baseURL string, key string, insecure bool) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{}
	}
	t.TLSClientConfig.InsecureSkipVerify = insecure

	authed := &http.Client{
		Transport: &authTripper{
			RoundTripper: t,
			key:          key,
		},
	}

	log.Printf("[DEBUG] using %s", baseURL)

	c := &Client{
		client: graphql.NewClient(u.String(), graphql.WithHTTPClient(authed)),
	}

	return c, c.Verify()
}
