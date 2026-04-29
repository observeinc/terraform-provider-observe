package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
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
		tokenHTTPClient := &http.Client{
			Timeout:   c.HTTPClientTimeout,
			Transport: transport,
		}

		if c.OAuth2.ClientSecret != "" {
			log.Printf("[INFO] Using OAuth2 Client Credentials authentication")
			cc := &clientcredentials.Config{
				ClientID:     c.OAuth2.ClientID,
				ClientSecret: c.OAuth2.ClientSecret,
				TokenURL:     c.OAuth2.TokenURL,
				Scopes:       c.OAuth2.Scopes,
			}
			tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient, tokenHTTPClient)
			client.tokenSource = cc.TokenSource(tokenCtx)
		} else {
			log.Printf("[INFO] Using OIDC authentication")
			client.tokenSource = oauth2.ReuseTokenSource(nil, &oidcTokenSource{
				cfg:        c,
				httpClient: tokenHTTPClient,
			})
		}
	}

	httpClient.Transport = client.withMiddleware(transport)
	return client, nil
}

type oidcTokenSource struct {
	cfg        *Config
	httpClient *http.Client
}

func (s *oidcTokenSource) Token() (*oauth2.Token, error) {
	oidcToken, err := s.fetchOIDCToken()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC token: %w", err)
	}

	cc := &clientcredentials.Config{
		ClientID: s.cfg.OAuth2.ClientID,
		TokenURL: s.cfg.OAuth2.TokenURL,
		Scopes:   s.cfg.OAuth2.Scopes,
		EndpointParams: url.Values{
			"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
			"client_assertion":      {oidcToken},
		},
		AuthStyle: oauth2.AuthStyleInParams,
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, s.httpClient)
	ts := cc.TokenSource(ctx)
	return ts.Token()
}

func (s *oidcTokenSource) fetchOIDCToken() (string, error) {
	if s.cfg.OAuth2.OIDCToken != "" {
		return s.cfg.OAuth2.OIDCToken, nil
	}
	if s.cfg.OAuth2.OIDCTokenFilePath != "" {
		b, err := os.ReadFile(s.cfg.OAuth2.OIDCTokenFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read OIDC token file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	if reqURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"); reqURL != "" {
		reqToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
		if reqToken == "" {
			return "", fmt.Errorf("ACTIONS_ID_TOKEN_REQUEST_TOKEN not set but ACTIONS_ID_TOKEN_REQUEST_URL is")
		}
		if aud := s.cfg.OAuth2.OIDCAudience; aud != "" {
			u, err := url.Parse(reqURL)
			if err != nil {
				return "", fmt.Errorf("failed to parse ACTIONS_ID_TOKEN_REQUEST_URL: %w", err)
			}
			q := u.Query()
			q.Set("audience", aud)
			u.RawQuery = q.Encode()
			reqURL = u.String()
		}
		req, err := http.NewRequestWithContext(context.Background(), "GET", reqURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create GHA request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+reqToken)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to make request to ACTIONS_ID_TOKEN_REQUEST_URL: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("github actions OIDC request returned status %d", resp.StatusCode)
		}
		var res struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return "", fmt.Errorf("failed to decode github actions OIDC response: %w", err)
		}
		return res.Value, nil
	}
	if tfcToken := os.Getenv("TFC_WORKLOAD_IDENTITY_TOKEN"); tfcToken != "" {
		return tfcToken, nil
	}
	return "", fmt.Errorf("no OIDC token source found (checked oidc_token, oidc_token_file_path, GitHub Actions, Terraform Cloud). If you intended to use standard OAuth2 Client Credentials, ensure client_secret is set")
}
