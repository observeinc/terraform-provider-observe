package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/observeinc/terraform-provider-observe/client/internal/customer"
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

var (
	// ErrUnauthorized is returned on 401
	ErrUnauthorized = errors.New("authorization error")
	defaultDomain   = "observeinc.com"
)

// Client implements a grossly simplified API client for Observe
type Client struct {
	customerID string
	domain     string
	token      string
	proxy      string
	insecure   bool
	userAgent  string
	flags      map[string]bool

	// our API does not allow concurrent FK creation, so we use a lock as a workaround
	obs2110 sync.Mutex

	httpClient *http.Client

	metaAPI     *meta.Client
	customerAPI *customer.Client
}

// login using whatever HTTP client we've assembled so far
func (c *Client) login(user string, password string) (string, error) {
	api := customer.New(c.getURL(""), c.httpClient)
	return api.Login(user, password)
}

// recursively unwrap error to figure out if it is temporary
func isTemporary(err error) bool {
	if t, ok := err.(net.Error); ok {
		return t.Temporary()
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return isTemporary(unwrapped)
	}
	return false
}

func NewClient(customerID string, options ...Option) (*Client, error) {
	c := &Client{
		customerID: customerID,
		domain:     defaultDomain,
		flags:      make(map[string]bool),
		httpClient: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		},
	}

	for _, o := range options {
		if err := o(c); err != nil {
			return nil, fmt.Errorf("failed to configure client: %w", err)
		}
	}

	// raise any unexpected status code from API as error
	wrapped := c.httpClient.Transport
	c.httpClient.Transport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.Host = c.getHost()
		resp, err := wrapped.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		switch resp.StatusCode {
		case http.StatusOK, http.StatusUnprocessableEntity:
			return resp, nil
		case http.StatusUnauthorized:
			return nil, ErrUnauthorized
		default:
			return nil, fmt.Errorf("received unexpected status code %d", resp.StatusCode)
		}
	})

	c.metaAPI = meta.New(c.getURL("/v1/meta"), c.httpClient)
	c.customerAPI = customer.New(c.getURL(""), c.httpClient)
	return c, c.metaAPI.Verify()
}

func (c *Client) getHost() string {
	return fmt.Sprintf("%s.%s", c.customerID, c.domain)
}

func (c *Client) getURL(path string) string {
	if c.proxy != "" {
		return fmt.Sprintf("%s%s", c.proxy, path)
	}
	return fmt.Sprintf("https://%s%s", c.getHost(), path)
}
