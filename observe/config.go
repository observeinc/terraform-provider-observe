package observe

import (
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
}

// Client returns an instantiated api client
func (c *Config) Client(userAgent string) (*client.Client, error) {
	var options []client.Option

	options = append(options, client.WithUserAgent(userAgent))

	if c.Domain != "" {
		options = append(options, client.WithDomain(c.Domain))
	}

	if c.Insecure {
		options = append(options, client.WithInsecure())
	}

	if c.UserEmail != "" {
		options = append(options, client.WithUserCredentials(c.UserEmail, c.UserPassword))
	}

	if c.Token != "" {
		options = append(options, client.WithToken(c.Token))
	}

	return client.NewClient(c.CustomerID, options...)
}
