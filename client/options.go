package client

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"crypto/tls"
	"net/url"
)

type Option func(*Client) error

// RoundTripperFunc implements http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (r RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

// WithProxy sets URL requests should be proxied through
func WithProxy(proxy string) Option {
	return func(c *Client) error {
		c.proxy = proxy
		_, err := url.Parse(c.proxy)
		return err
	}
}

// WithDomain overrides domain name used
func WithDomain(domain string) Option {
	return func(c *Client) error {
		c.domain = domain
		_, err := url.Parse(fmt.Sprintf("https://%s.%s", c.customerID, domain))
		return err
	}
}

// WithHTTPClient overrides default HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if c.httpClient != nil {
			return fmt.Errorf("client already set")
		}
		c.httpClient = client
		return nil
	}
}

// WithInsecure sets HTTP client to not verify TLS requests
// Must be set after WithHTTPClient, if both options are present
func WithInsecure() Option {
	return func(c *Client) error {
		t, ok := c.httpClient.Transport.(*http.Transport)
		if !ok {
			return fmt.Errorf("failed to configure TLS client")
		}
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.InsecureSkipVerify = true
		return nil
	}
}

// WithToken sets bearer token
func WithToken(token string) Option {
	return func(c *Client) error {
		wrapped := c.httpClient.Transport
		c.httpClient.Transport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s %s", c.customerID, token))
			return wrapped.RoundTrip(req)
		})
		return nil
	}
}

// WithUserCredentials users regular login flow to generate token
func WithUserCredentials(user, password string) Option {
	return func(c *Client) (err error) {
		token, err := c.login(user, password)
		if err != nil {
			return err
		}
		return WithToken(token)(c)
	}
}

// WithUserAgent sets user agent on all requests
func WithUserAgent(userAgent string) Option {
	return func(c *Client) (err error) {
		wrapped := c.httpClient.Transport
		c.httpClient.Transport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("User-Agent", userAgent)
			return wrapped.RoundTrip(req)
		})
		return nil
	}
}

// WithRetry enables retry on temporary network failures
func WithRetry(count int, wait time.Duration) Option {
	return func(c *Client) (err error) {
		wrapped := c.httpClient.Transport
		c.httpClient.Transport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := wrapped.RoundTrip(req)
			for retry := 0; err != nil && isTemporary(err) && retry < count; retry++ {
				log.Printf("[WARN] request failed with temporary error: %s\n", err)
				time.Sleep(wait)
				log.Printf("[WARN] attempting recovery (%d/%d)\n", retry+1, count)
				resp, err = wrapped.RoundTrip(req)
			}
			return resp, err
		})
		return nil
	}
}

// WithLogging enables logging of requests and responses
func WithLogging(dumpRequestBody, dumpResponseBody bool) Option {
	return func(c *Client) (err error) {
		wrapped := c.httpClient.Transport
		c.httpClient.Transport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			s, err := httputil.DumpRequest(req, dumpRequestBody)
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] %s\n", s)

			resp, err := wrapped.RoundTrip(req)

			if resp != nil {
				s, _ = httputil.DumpResponse(resp, dumpResponseBody)
				log.Printf("[DEBUG] %s\n", s)
			}

			return resp, err
		})
		return nil
	}
}

// WithFlags enables feature flags. Multiple calls will overwrite previous settings
func WithFlags(flags map[string]bool) Option {
	return func(c *Client) (err error) {
		for k, v := range flags {
			c.flags[k] = v
		}
		return nil
	}
}
