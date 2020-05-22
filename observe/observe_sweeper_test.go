package observe

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/acctest"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func init() {
	acctest.UseBinaryDriver("observe", Provider)
}

func sharedClient() (*observe.Client, error) {
	c := &Config{
		CustomerID:   os.Getenv("OBSERVE_CUSTOMER"),
		UserEmail:    os.Getenv("OBSERVE_USER_EMAIL"),
		UserPassword: os.Getenv("OBSERVE_USER_PASSWORD"),
		Token:        os.Getenv("OBSERVE_TOKEN"),
		Domain:       os.Getenv("OBSERVE_DOMAIN"),
	}
	return c.Client()
}
