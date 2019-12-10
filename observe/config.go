package observe

import (
	"github.com/observeinc/terraform-provider-observe/client"
)

type Config struct {
	BaseURL string
	ApiKey  string
}

func (c *Config) Client() (*client.Client, error) {
	return client.NewClient(c.BaseURL, c.ApiKey)
}
