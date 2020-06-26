package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/observeinc/terraform-provider-observe/client/internal/api"
)

var (
	// ErrUnauthorized is returned on 401
	ErrUnauthorized = errors.New("authorization error")
	defaultDomain   = "observeinc.com"
)

type authTripper struct {
	http.RoundTripper
	key       string
	userAgent string
}

func (t *authTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// log request before adding authorization header
	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}

	s, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] %s\n", s)

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

	s, _ = httputil.DumpResponse(resp, true)
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

// Client implements a grossly simplified API client for Observe
type Client struct {
	customerID string
	domain     string
	token      string
	insecure   bool
	userAgent  string

	httpClient *http.Client
	gqlClient  *graphql.Client
	api        *api.Client
}

// Verify checks if we can connect to API.
func (c *Client) Verify() error {
	req := graphql.NewRequest(`{ currentUser { id } }`)
	var respData interface{}
	if err := c.gqlClient.Run(context.Background(), req, &respData); err != nil {
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
	err := c.gqlClient.Run(context.Background(), req, &result)
	return result, err
}

func NewClient(customerID string, options ...Option) (*Client, error) {
	c := &Client{
		customerID: customerID,
		domain:     defaultDomain,
		httpClient: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		},
	}

	for _, o := range options {
		if err := o(c); err != nil {
			return nil, fmt.Errorf("failed to configure client: %w", err)
		}
	}

	if c.token != "" {
		wrapped := c.httpClient.Transport
		c.httpClient.Transport = &authTripper{
			RoundTripper: wrapped,
			key:          fmt.Sprintf("%s %s", c.customerID, c.token),
			userAgent:    c.userAgent,
		}
	}

	gqlURL := fmt.Sprintf("https://%s.%s/v1/meta", c.customerID, c.domain)
	c.gqlClient = graphql.NewClient(gqlURL, graphql.WithHTTPClient(c.httpClient))
	c.api = api.New(c)
	return c, c.Verify()
}

func (c *Client) login(user, password string) (string, error) {
	var result struct {
		AccessKey string `json:"access_key"`
		Ok        bool   `json:"ok"`
	}

	err := c.do("POST", "/v1/login", map[string]interface{}{
		"user_email":    user,
		"user_password": password,
	}, &result)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}

	return result.AccessKey, nil
}

// do is a helper to run HTTP request
func (c *Client) do(method string, path string, body map[string]interface{}, result interface{}) error {

	var (
		endpoint = fmt.Sprintf("https://%s.%s%s", c.customerID, c.domain, path)
		reqBody  io.Reader
	)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if s, err := httputil.DumpRequest(req, false); err != nil {
		return err
	} else {
		log.Printf("[DEBUG] %s\n", s)
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
