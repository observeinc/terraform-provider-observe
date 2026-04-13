package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/observeinc/terraform-provider-observe/client/internal/collect"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/rest"
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

	tokenSource oauth2.TokenSource

	Meta    *meta.Client
	Rest    *rest.Client
	Collect *collect.Client

	resolveWorkspace   sync.Once
	cachedWorkspaceID  string
	cachedWorkspaceErr error
}

// login to retrieve a valid token, only need to do this once
func (c *Client) loginOnFirstRun(ctx context.Context) (loginErr error) {
	if requiresAuth(ctx) && c.ApiToken == nil && c.UserEmail != nil {
		c.login.Do(func() {
			ctx = setSensitive(ctx, true)
			ctx = requireAuth(ctx, false)

			token, err := c.Rest.Login(ctx, *c.UserEmail, *c.UserPassword)
			if err != nil {
				loginErr = fmt.Errorf("failed to retrieve token: %w", err)
			} else {
				c.ApiToken = &token
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

func shouldRetryRequest(resp *http.Response, err error) bool {
	if err != nil {
		return isTemporary(err)
	}
	return resp != nil && resp.StatusCode == http.StatusTooManyRequests
}

func isTemporary(err error) bool {
	if t, ok := err.(net.Error); ok {
		return t.Temporary()
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return isTemporary(unwrapped)
	}
	return false
}

// setTrace adds logging info to every outbound request
func (c *Client) setTrace(req *http.Request) *http.Request {
	trace := &httptrace.ClientTrace{
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			log.Printf("[TRACE] DNS Info: %+v\n", dnsInfo)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			log.Printf("[TRACE] Got Conn: %+v\n", connInfo)
		},
	}
	return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
}

// withMiddelware adds logging, auth handling to all outgoing requests
func (c *Client) withMiddleware(wrapped http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
		ctx := req.Context()

		if c.UserAgent != nil {
			req.Header.Set("User-Agent", *c.UserAgent)
		}
		if c.TraceParent != nil {
			req.Header.Set("Traceparent", *c.TraceParent)
		}

		// log request and response
		c.logRequest(ctx, req)
		defer func() {
			c.logResponse(ctx, resp)
		}()

		// obtain token if needed
		if c.tokenSource != nil {
			tok, err := c.tokenSource.Token()
			if err != nil {
				return nil, fmt.Errorf("failed to obtain oauth2 token: %w", err)
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s %s", c.CustomerID, tok.AccessToken))
		} else {
			if err := c.loginOnFirstRun(ctx); err != nil {
				return nil, fmt.Errorf("failed to login: %w", err)
			}
			if c.ApiToken != nil {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s %s", c.CustomerID, *c.ApiToken))
			}
		}

		resp, err = wrapped.RoundTrip(c.setTrace(req))
		waitBeforeRetry := c.RetryWait
		for retry := 0; shouldRetryRequest(resp, err) && retry < c.RetryCount; retry++ {
			log.Printf("[WARN] retryable request error, retrying in %s (%d/%d)\n", waitBeforeRetry, retry+1, c.RetryCount)
			time.Sleep(waitBeforeRetry)
			waitBeforeRetry *= 2
			if waitBeforeRetry > meta.MaxRetryBackoff {
				waitBeforeRetry = meta.MaxRetryBackoff
			}
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

	metaAPI, err := meta.New(customerURL+"/v1/meta", httpClient, meta.WithRetries(c.RetryCount, c.RetryWait))
	if err != nil {
		return nil, fmt.Errorf("failed to configure meta API: %w", err)
	}

	client := &Client{
		Config:  c,
		Meta:    metaAPI,
		Rest:    rest.New(customerURL, httpClient),
		Collect: collectAPI,
	}

	if c.OAuth2 != nil {
		cc := &clientcredentials.Config{
			ClientID:     c.OAuth2.ClientID,
			ClientSecret: c.OAuth2.ClientSecret,
			TokenURL:     c.OAuth2.TokenURL,
			Scopes:       c.OAuth2.Scopes,
		}
		tokenHTTPClient := &http.Client{
			Timeout:   c.HTTPClientTimeout,
			Transport: transport,
		}
		tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient, tokenHTTPClient)
		client.tokenSource = cc.TokenSource(tokenCtx)
	}

	httpClient.Transport = client.withMiddleware(transport)
	return client, nil
}
