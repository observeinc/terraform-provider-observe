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
	resource.AddTestSweepers("observe_poller_sweep", &resource.Sweeper{
		Name: "observe_poller_sweep",
		F:    pollerSweeperFunc,
	})
	resource.AddTestSweepers("observe_datastream_sweep", &resource.Sweeper{
		Name: "observe_datastream_sweep",
		F:    datastreamSweeperFunc,
	})
	resource.AddTestSweepers("observe_folder_sweep", &resource.Sweeper{
		Name: "observe_folder_sweep",
		F:    folderSweeperFunc,
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

	if token := os.Getenv("OBSERVE_API_TOKEN"); token != "" {
		config.ApiToken = &token
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
		for _, ds := range workspace.Datasets {
			if prefixRe.MatchString(ds.Label) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", ds.Label, ds.Id)
				if err := client.DeleteDataset(ctx, ds.Id); err != nil {
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
			"workspaceId": workspace.Id,
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

func pollerSweeperFunc(s string) error {
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
		query pollers($workspaceId: ObjectId!) {
			pollers(workspaceId: $workspaceId) {
				id
				config {
					name
				}
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup pollers: %w", err)
		}

		for _, i := range result["pollers"].([]interface{}) {
			var (
				item   = i.(map[string]interface{})
				id     = item["id"].(string)
				config = item["config"].(map[string]interface{})
			)
			name := config["name"].(string)
			if prefixRe.MatchString(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeletePoller(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func datastreamSweeperFunc(s string) error {
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
		query datastreams($workspaceId: ObjectId!) {
			datastreams(workspaceId: $workspaceId) {
				id
				name
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup datastreams: %w", err)
		}

		for _, i := range result["datastreams"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				id   = item["id"].(string)
				name = item["name"].(string)
			)
			if prefixRe.MatchString(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteDatastream(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func folderSweeperFunc(s string) error {
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
		query folders($workspaceId: ObjectId!) {
			folders(workspaceId: $workspaceId) {
				id
				name
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup folders: %w", err)
		}

		for _, i := range result["folders"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				id   = item["id"].(string)
				name = item["name"].(string)
			)
			if prefixRe.MatchString(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteFolder(ctx, id); err != nil {
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
