package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"
)

var (
	ErrUnauthorized         = errors.New("authorization error")
	ErrMissingCustomer      = errors.New("customer ID not set")
	ErrMissingDomain        = errors.New("domain not set")
	ErrTokenEmail           = errors.New("token and user email are mutually exclusive")
	ErrMissingPassword      = errors.New("password must be set when user email is provided")
	ErrMissingRetryDuration = errors.New("retry duration must be larger than 0")
	ErrMalformedSource      = errors.New("source identifier must follow \"category/comment\" format")
)

// Config contains all configuration attributes for our client.
//
// Clients sharing a same config may end up using the same underlying HTTP
// client in order to reuse connections more efficiently. We use Hash() to
// cache clients. Do not introduce private fields to this struct without
// adjusting how the hash function is computed.
type Config struct {
	CustomerID string `json:"customer_id"`
	Domain     string `json:"domain"`

	// auth
	UserAgent    *string `json:"user_agent"`
	ApiToken     *string `json:"api_token"`
	UserEmail    *string `json:"user_email"`
	UserPassword *string `json:"user_password"`

	// client options
	Insecure bool `json:"insecure"`

	RetryCount int           `json:"retry_count"`
	RetryWait  time.Duration `json:"retry_wait"`

	HTTPClientTimeout time.Duration `json:"http_timeout"`
	Flags             map[string]bool

	// optional source identifier when managing Observe resources
	Source *string `json:"source"`

	// optional managing id to tag Observe resources with
	ManagingObjectID *string `json:"managing_object_id"`

	// optional traceparent identifier to pass via header
	TraceParent *string `json:"traceparent"`

	// enable extra queries needed to export bindings
	ExportObjectBindings bool `json:"export_object_bindings"`

	// Allow setting default materialization mode for dataset resources
	DefaultRematerializationMode *string `json:"default_rematerialization_mode"`

	// Skip making dry run API requests for dataset changes during the plan stage (for validation)
	SkipDatasetDryRuns bool `json:"skip_dataset_dry_runs"`
}

func (c *Config) Hash() uint64 {
	v, err := hashstructure.Hash(c, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to hash configuration: %s", err))
	}
	return v
}

func (c *Config) Validate() error {
	if c.CustomerID == "" {
		return ErrMissingCustomer
	}

	if c.Domain == "" {
		return ErrMissingDomain
	}

	if c.ApiToken != nil && c.UserEmail != nil {
		return ErrTokenEmail
	}

	if c.UserEmail != nil && c.UserPassword == nil {
		return ErrMissingPassword
	}

	if c.RetryCount > 0 && c.RetryWait == time.Duration(0) {
		return ErrMissingRetryDuration
	}

	if c.Source != nil && !strings.Contains(*c.Source, "/") {
		return ErrMalformedSource
	}

	return nil
}
