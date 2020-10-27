package observe

import (
	"context"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func init() {
	resource.AddTestSweepers("observe_dataset_sweep", &resource.Sweeper{
		Name: "observe_dataset_sweep",
		F:    datasetSweeperFunc,
	})
}

func sharedClient(s string) (*observe.Client, error) {
	c := &Config{
		CustomerID:   os.Getenv("OBSERVE_CUSTOMER"),
		UserEmail:    os.Getenv("OBSERVE_USER_EMAIL"),
		UserPassword: os.Getenv("OBSERVE_USER_PASSWORD"),
		Token:        os.Getenv("OBSERVE_TOKEN"),
		Domain:       os.Getenv("OBSERVE_DOMAIN"),
		Insecure:     os.Getenv("OBSERVE_INSECURE") == "true",
	}
	return c.Client()
}

var (
	prefixRe = regexp.MustCompile(`^tf-\d{19,}`)
)

func datasetSweeperFunc(s string) error {
	client, err := sharedClient(s)
	if err != nil {
		return err
	}

	ctx := context.Background()

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	for _, workspace := range workspaces {
		for name, id := range workspace.Datasets {
			if prefixRe.MatchString(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteDataset(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}
