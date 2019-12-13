package observe

import (
	"fmt"
	"github.com/observeinc/terraform-provider-observe/client"
)

type Config struct {
	CustomerID string
	Token      string
	Domain     string
}

func (c *Config) Client() (*client.Client, error) {
	var (
		baseURL = fmt.Sprintf("https://%s.%s/v1/meta", c.CustomerID, c.Domain)
		key     = fmt.Sprintf("%s %s", c.CustomerID, c.Token)
	)
	return client.NewClient(baseURL, key)
}
