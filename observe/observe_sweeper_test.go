package observe

import (
	"context"
	"fmt"
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
	resource.AddTestSweepers("observe_monitor_sweep", &resource.Sweeper{
		Name: "observe_monitor_sweep",
		F:    monitorSweeperFunc,
	})
}

func sharedClient(s string) (*observe.Client, error) {
	config := &observe.Config{
		CustomerID: os.Getenv("OBSERVE_CUSTOMER"),
		Domain:     os.Getenv("OBSERVE_DOMAIN"),
		Insecure:   os.Getenv("OBSERVE_INSECURE") == "true",
	}

	if userEmail := os.Getenv("OBSERVE_USER_EMAIL"); userEmail != "" {
		userPassword := os.Getenv("OBSERVE_USER_PASSWORD")
		config.UserEmail = &userEmail
		config.UserPassword = &userPassword
	}

	if token := os.Getenv("OBSERVE_TOKEN"); token != "" {
		config.Token = &token
	}

	return observe.New(config)
}

var (
	prefixRe = regexp.MustCompile(`^tf-\d{16,}`)
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

func monitorSweeperFunc(s string) error {
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
		result, err := client.Meta.Run(ctx, `
		query getMonitorsInWorkspace($workspaceId: ObjectId!) {
			monitorsInWorkspace(workspaceId: $workspaceId) {
				id
				name
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.ID,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup monitors: %w", err)
		}

		for _, i := range result["monitorsInWorkspace"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				name = item["name"].(string)
				id   = item["id"].(string)
			)
			if prefixRe.MatchString(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteMonitor(ctx, id); err != nil {
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
