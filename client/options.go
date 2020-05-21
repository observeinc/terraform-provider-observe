package client

import (
	"fmt"
	"net/http"

	"crypto/tls"
	"net/url"
)

type Option func(*Client) error

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
		c.token = token
		return nil
	}
}

func WithUserCredentials(user, password string) Option {
	return func(c *Client) (err error) {
		c.token, err = c.login(user, password)
		return err
	}
}
