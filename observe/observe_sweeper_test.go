package observe

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedClient() (*observe.Client, error) {
	c := &Config{
		CustomerID: os.Getenv("OBSERVE_CUSTOMER"),
		Token:      os.Getenv("OBSERVE_TOKEN"),
		Domain:     os.Getenv("OBSERVE_DOMAIN"),
	}
	return c.Client()
}
