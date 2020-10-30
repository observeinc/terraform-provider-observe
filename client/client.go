package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/collect"
	"github.com/observeinc/terraform-provider-observe/client/internal/customer"
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

// RoundTripperFunc implements http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (r RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

// Client handles interacting with our API(s)
type Client struct {
	*Config

	login sync.Once

	// our API does not allow concurrent FK creation, so we use a lock as a workaround
	obs2110 sync.Mutex

	Meta     *meta.Client
	Customer *customer.Client
	Collect  *collect.Client
}

// login to retrieve a valid token, only need to do this once
func (c *Client) loginOnFirstRun(ctx context.Context) (loginErr error) {
	if requiresAuth(ctx) && c.Token == nil && c.UserEmail != nil {
		c.login.Do(func() {
			ctx = setSensitive(ctx, true)
			ctx = requireAuth(ctx, false)

			token, err := c.Customer.Login(ctx, *c.UserEmail, *c.UserPassword)
			if err != nil {
				loginErr = fmt.Errorf("failed to retrieve token: %w", err)
			} else {
				c.Token = &token
			}
		})
	}
	return
}

func (c *Client) logRequest(ctx context.Context, req *http.Request) {
	sensitive := isSensitive(ctx)
	if sensitive {
		log.Printf("[DEBUG] sensitive payload, omitting request body")
	}

	s, err := httputil.DumpRequest(req, !sensitive)
	if err != nil {
		log.Printf("[WARN] failed to dump request: %s\n", err)
	}
	log.Printf("[DEBUG] %s\n", s)
}

func (c *Client) logResponse(ctx context.Context, resp *http.Response) {
	if resp != nil {
		s, _ := httputil.DumpResponse(resp, !isSensitive(ctx))
		log.Printf("[DEBUG] %s\n", s)
	}
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

// withMiddelware adds logging, auth handling to all outgoing requests
func (c *Client) withMiddleware(wrapped http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
		ctx := req.Context()

		if c.UserAgent != nil {
			req.Header.Set("User-Agent", *c.UserAgent)
		}

		// log request and response
		c.logRequest(ctx, req)
		defer func() {
			c.logResponse(ctx, resp)
		}()

		// obtain token if needed - only first request requiring auth will login
		if err := c.loginOnFirstRun(ctx); err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}

		// set auth header only after having logged request
		if c.Token != nil {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s %s", c.CustomerID, *c.Token))
		}

		resp, err = wrapped.RoundTrip(req)
		waitBeforeRetry := c.RetryWait
		for retry := 0; err != nil && isTemporary(err) && retry < c.RetryCount; retry++ {
			log.Printf("[WARN] request failed with temporary error: %s\n", err)
			time.Sleep(waitBeforeRetry)
			waitBeforeRetry += c.RetryWait
			log.Printf("[WARN] attempting recovery (%d/%d)\n", retry+1, c.RetryCount)
			resp, err = wrapped.RoundTrip(req)
		}
		return
	})
}

// New returns a new client
func New(c *Config) (*Client, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}

	// first we must create an HTTP client for all subsequent requests
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.Proxy != nil {
		proxyURL, _ := url.Parse(*c.Proxy)
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// disable TLS verification if necessary
	if c.Insecure {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// create APIs
	httpClient := &http.Client{Timeout: c.HTTPClientTimeout}

	customerURL := fmt.Sprintf("https://%s.%s", c.CustomerID, c.Domain)
	collectURL := fmt.Sprintf("https://collect.%s", c.Domain)

	collectAPI, err := collect.New(collectURL, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to configure collect API: %w", err)
	}

	metaAPI, err := meta.New(customerURL+"/v1/meta", httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to configure meta API: %w", err)
	}

	client := &Client{
		Config:   c,
		Meta:     metaAPI,
		Customer: customer.New(customerURL, httpClient),
		Collect:  collectAPI,
	}

	httpClient.Transport = client.withMiddleware(transport)
	return client, nil
}
