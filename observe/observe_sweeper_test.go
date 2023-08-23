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
	resource.AddTestSweepers("observe_workspace", &resource.Sweeper{
		Name: "observe_workspace",
		F:    workspaceSweeperFunc,
		Dependencies: []string{
			"observe_dataset",
			"observe_monitor",
			"observe_poller",
			"observe_datastream",
			"observe_folder",
			"observe_preferred_path",
			"observe_bookmark_group",
			"observe_worksheet",
			"observe_app",
			"observe_rbac_statement",
		},
	})
	resource.AddTestSweepers("observe_dataset", &resource.Sweeper{
		Name: "observe_dataset",
		F:    datasetSweeperFunc,
		Dependencies: []string{
			"observe_preferred_path",
			"observe_datastream",
			"observe_app",
		},
	})
	resource.AddTestSweepers("observe_monitor", &resource.Sweeper{
		Name: "observe_monitor",
		F:    monitorSweeperFunc,
		Dependencies: []string{
			"observe_app",
		},
	})
	resource.AddTestSweepers("observe_poller", &resource.Sweeper{
		Name: "observe_poller",
		F:    pollerSweeperFunc,
		Dependencies: []string{
			"observe_app",
		},
	})
	resource.AddTestSweepers("observe_datastream", &resource.Sweeper{
		Name: "observe_datastream",
		F:    datastreamSweeperFunc,
		Dependencies: []string{
			"observe_poller",
			"observe_app",
			"observe_filedrop",
		},
	})
	resource.AddTestSweepers("observe_folder", &resource.Sweeper{
		Name: "observe_folder",
		F:    folderSweeperFunc,
		Dependencies: []string{
			"observe_preferred_path",
			"observe_app",
		},
	})
	resource.AddTestSweepers("observe_preferred_path", &resource.Sweeper{
		Name: "observe_preferred_path",
		F:    preferredPathSweeperFunc,
	})
	resource.AddTestSweepers("observe_bookmark_group", &resource.Sweeper{
		Name: "observe_bookmark_group",
		F:    bookmarkGroupSweeperFunc,
	})
	resource.AddTestSweepers("observe_worksheet", &resource.Sweeper{
		Name: "observe_worksheet",
		F:    worksheetSweeperFunc,
	})
	resource.AddTestSweepers("observe_app", &resource.Sweeper{
		Name: "observe_app",
		F:    appSweeperFunc,
	})
	resource.AddTestSweepers("observe_rbac_statement", &resource.Sweeper{
		Name: "observe_rbac_statement",
		F:    rbacStatementSweeperFunc,
	})
	resource.AddTestSweepers("observe_rbac_group", &resource.Sweeper{
		Name: "rbac_group",
		F:    rbacGroupSweeperFunc,
		Dependencies: []string{
			"observe_rbac_statement",
		},
	})
	resource.AddTestSweepers("observe_filedrop", &resource.Sweeper{
		Name: "observe_filedrop",
		F:    filedropSweeperFunc,
	})
}

type client struct {
	*observe.Client
	MatchName func(string) bool
}

func sharedClient(pattern string) (*client, error) {
	patternRe, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile name pattern: %s", err)
	}

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

	oclient, err := observe.New(config)
	if err != nil {
		return nil, err
	}

	return &client{
		Client: oclient,
		MatchName: func(s string) bool {
			fmt.Printf("matching %s %v %s\n", s, patternRe.MatchString(s), pattern)
			return patternRe.MatchString(s)
		},
	}, nil
}

func workspaceSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
	if err != nil {
		return err
	}

	ctx := context.Background()

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	for _, workspace := range workspaces {
		if client.MatchName(workspace.Label) {
			log.Printf("[WARN] Deleting %s [id=%s]\n", workspace.Label, workspace.Id)
			if err := client.DeleteWorkspace(ctx, workspace.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func datasetSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query getDatasetsInWorkspace($workspaceId: ObjectId!) {
		    workspace(id: $workspaceId) {
			    datasets {
			        id
			        label
			        managedById
			    }
		    }
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup datasets: %w", err)
		}

		result = result["workspace"].(map[string]interface{})
		for _, i := range result["datasets"].([]interface{}) {
			var (
				item        = i.(map[string]interface{})
				label       = item["label"].(string)
				id          = item["id"].(string)
				managedById = item["managedById"]
			)
			if client.MatchName(label) && managedById == nil {
				log.Printf("[WARN] Deleting %s [id=%s]\n", label, id)
				if err := client.DeleteDataset(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func monitorSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteMonitor(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func pollerSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeletePoller(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func datastreamSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteDatastream(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func folderSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteFolder(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func preferredPathSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query preferredPaths($workspaceId: ObjectId!) {
			preferredPathSearch(terms: {workspaceId: [$workspaceId]}) {
				results {
					preferredPath {
						id
						name
					}
				}
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup preferred paths: %w", err)
		}

		result = result["preferredPathSearch"].(map[string]interface{})

		for _, i := range result["results"].([]interface{}) {
			var (
				result = i.(map[string]interface{})
				item   = result["preferredPath"].(map[string]interface{})

				id   = item["id"].(string)
				name = item["name"].(string)
			)
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeletePreferredPath(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func bookmarkGroupSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query bookmarkGroups($workspaceId: ObjectId!) {
			searchBookmarkGroups(workspaceIds:[$workspaceId]) {
				id
				name
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup bookmark groups: %w", err)
		}

		for _, i := range result["searchBookmarkGroups"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				id   = item["id"].(string)
				name = item["name"].(string)
			)
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting bookmark group %s [id=%s]\n", name, id)
				// Deleting a bookmark group will delete all bookmarks in it
				if err := client.DeleteBookmarkGroup(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func worksheetSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query worksheets($workspaceId: ObjectId!) {
			worksheetSearch(terms: {workspaceId: [$workspaceId]}) {
				worksheets {
					worksheet {
						id
						name
					}
				}
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup worksheets: %w", err)
		}

		for _, i := range result["worksheetSearch"].(map[string]interface{})["worksheets"].([]interface{}) {
			var (
				ws   = i.(map[string]interface{})["worksheet"].(map[string]interface{})
				id   = ws["id"].(string)
				name = ws["name"].(string)
			)
			if client.MatchName(name) {
				log.Printf("[WARN] Deleting %s [id=%s]\n", name, id)
				if err := client.DeleteWorksheet(ctx, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func appSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query apps($workspaceId: ObjectId!) {
			apps(workspaceId: $workspaceId) {
				id
				name
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup apps: %w", err)
		}

		for _, i := range result["apps"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				id   = item["id"].(string)
				name = item["name"].(string)
			)

			log.Printf("[WARN] Deleting app %s [id=%s]\n", name, id)
			if err := client.DeleteApp(ctx, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func rbacGroupSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := client.Meta.Run(ctx, `
	query rbacGroups {
		rbacGroups {
			id
			name
		}
	}`, nil)

	if err != nil {
		return fmt.Errorf("failed to lookup rbac groups: %w", err)
	}

	for _, i := range result["rbacGroups"].([]interface{}) {
		var (
			item = i.(map[string]interface{})
			id   = item["id"].(string)
			name = item["name"].(string)
		)

		if client.MatchName(name) {
			log.Printf("[WARN] Deleting rbac group %s [id=%s]\n", name, id)
			if err := client.DeleteRbacGroup(ctx, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func rbacStatementSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := client.Meta.Run(ctx, `
	query rbacStatements {
		rbacStatements {
			id
   			description
		}
	}`, nil)

	if err != nil {
		return fmt.Errorf("failed to lookup rbac statements: %w", err)
	}

	for _, i := range result["rbacStatements"].([]interface{}) {
		var (
			item        = i.(map[string]interface{})
			id          = item["id"].(string)
			description = item["description"].(string)
		)

		if client.MatchName(description) {
			log.Printf("[WARN] Deleting rbac statement [id=%s]\n", id)
			if err := client.DeleteRbacStatement(ctx, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func filedropSweeperFunc(pattern string) error {
	client, err := sharedClient(pattern)
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
		query filedrops($workspaceId: ObjectId!) {
			searchFiledrop(workspaceId: $workspaceId) {
				results {
					id
					name
				}
			}
		}`, map[string]interface{}{
			"workspaceId": workspace.Id,
		})

		if err != nil {
			return fmt.Errorf("failed to lookup filedrops: %w", err)
		}

		result = result["searchFiledrop"].(map[string]interface{})

		for _, i := range result["results"].([]interface{}) {
			var (
				item = i.(map[string]interface{})
				id   = item["id"].(string)
				name = item["name"].(string)
			)

			log.Printf("[WARN] Deleting filedrop %s [id=%s]\n", name, id)
			if err := client.DeleteFiledrop(ctx, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}
