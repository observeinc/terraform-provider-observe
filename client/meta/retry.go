package meta

import (
	"context"
	"log"
	"time"

	"github.com/Khan/genqlient/graphql"
)

const (
	ErrorCodeResourceExhausted = "RESOURCE_EXHAUSTED"
	MaxRetryBackoff            = 1 * time.Minute
)

var retryableErrorCodes = []string{
	ErrorCodeResourceExhausted,
}

type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
}

func isRetryable(err error) bool {
	for _, code := range retryableErrorCodes {
		if HasErrorCode(err, code) {
			return true
		}
	}
	return false
}

// withRetry calls fn and retries with exponential backoff when the returned
// error contains a retryable GraphQL error code.
func withRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	err := fn()
	if err == nil || !isRetryable(err) {
		return err
	}

	backoff := cfg.InitialBackoff
	for retry := 0; retry < cfg.MaxRetries; retry++ {
		log.Printf("[WARN] retryable GraphQL error, retrying in %s (%d/%d): %s",
			backoff, retry+1, cfg.MaxRetries, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		err = fn()
		if err == nil || !isRetryable(err) {
			return err
		}

		backoff *= 2
		if backoff > MaxRetryBackoff {
			backoff = MaxRetryBackoff
		}
	}

	return err
}

type retryClient struct {
	inner  graphql.Client
	config RetryConfig
}

// newRetryClient wraps a graphql.Client to automatically retry on retryable
// errors
func newRetryClient(inner graphql.Client, config RetryConfig) graphql.Client {
	if config.MaxRetries <= 0 {
		return inner
	}
	return &retryClient{inner: inner, config: config}
}

func (c *retryClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	return withRetry(ctx, c.config, func() error {
		resp.Errors = nil
		resp.Extensions = nil
		return c.inner.MakeRequest(ctx, req, resp)
	})
}
