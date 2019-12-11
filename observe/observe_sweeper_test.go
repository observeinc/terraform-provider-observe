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

func sharedClient(t *testing.T) *observe.Client {
	client, err := observe.NewClient(os.Getenv("OBSERVE_URL"), os.Getenv("OBSERVE_KEY"))
	if err != nil {
		t.Fatal("could not create client", err)
	}
	return client
}
