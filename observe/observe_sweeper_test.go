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
	client, err := observe.NewClient(os.Getenv("OBSERVE_URL"), os.Getenv("OBSERVE_TOKEN"))
	if err != nil {
		return client, err
	}

	return client, nil
}
