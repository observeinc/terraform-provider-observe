package observe

import (
	"time"

	"github.com/observeinc/terraform-provider-observe/client"
)

// Config for provider
type Config struct {
	CustomerID   string
	Token        string
	UserEmail    string
	UserPassword string
	Domain       string
	Insecure     bool
	UserAgent    string
	RetryCount   int
	RetryWait    time.Duration
}

// Client returns an instantiated api client
func (c *Config) Client() (*client.Client, error) {
	var options []client.Option

	if c.Insecure {
		options = append(options, client.WithInsecure())
	}

	if c.Domain != "" {
		options = append(options, client.WithDomain(c.Domain))
	}

	if c.UserAgent != "" {
		options = append(options, client.WithUserAgent(c.UserAgent))
	}

	if c.RetryCount > 0 {
		options = append(options, client.WithRetry(c.RetryCount, c.RetryWait))
	}

	if c.UserEmail != "" {
		options = append(options, client.WithUserCredentials(c.UserEmail, c.UserPassword))
	} else if c.Token != "" {
		options = append(options, client.WithToken(c.Token))
	}

	// we add logging after auth, that way we won't log the authorization header
	options = append(options, client.WithLogging(true, true))

	return client.NewClient(c.CustomerID, options...)
}
