package client

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

var (
	ErrUnauthorized         = errors.New("authorization error")
	ErrMissingCustomer      = errors.New("customer ID not set")
	ErrMissingDomain        = errors.New("domain not set")
	ErrTokenEmail           = errors.New("token and user email are mutually exclusive")
	ErrMissingPassword      = errors.New("password must be set when user email is provided")
	ErrMissingRetryDuration = errors.New("retry duration must be larger than 0")
)

type Config struct {
	CustomerID string `json:"customer_id"`
	Domain     string `json:"domain"`

	// auth
	UserAgent    *string `json:"user_agent"`
	Token        *string `json:"token"`
	UserEmail    *string `json:"user_email"`
	UserPassword *string `json:"user_password"`

	// client options
	Insecure bool    `json:"insecure"`
	Proxy    *string `json:"proxy"`

	RetryCount int           `json:"retry_count"`
	RetryWait  time.Duration `json:"retry_wait"`

	HTTPClientTimeout time.Duration `json:"http_timeout"`
	Flags             map[string]bool
}

func (c *Config) Validate() error {
	if c.CustomerID == "" {
		return ErrMissingCustomer
	}

	if c.Domain == "" {
		return ErrMissingDomain
	}

	if c.Token != nil && c.UserEmail != nil {
		return ErrTokenEmail
	}

	if c.UserEmail != nil && c.UserPassword == nil {
		return ErrMissingPassword
	}

	if c.Proxy != nil {
		if _, err := url.Parse(*c.Proxy); err != nil {
			return fmt.Errorf("failed to parse proxy URL: %w", err)
		}
	}

	if c.RetryCount > 0 && c.RetryWait == time.Duration(0) {
		return ErrMissingRetryDuration
	}

	return nil
}
