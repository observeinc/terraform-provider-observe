package observe

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedClient() (*Client, error) {
	client, err := NewClient(os.Getenv("OBSERVE_URL"), os.Getenv("OBSERVE_TOKEN"))
	if err != nil {
		return client, err
	}

	return client, nil
}
