package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccObserveIngestRoute(t *testing.T) {
	namePrefix := acctest.RandomWithPrefix("tf-ingest-route")
	initialConfig := testAccIngestRouteConfig(namePrefix, ingestRouteConfigOptions{
		primaryPipeline:           "filter true",
		primaryEnabled:            true,
		includePrimarySecondary:   true,
		ingestRouteOrderDirection: ingestRouteOrderPrimaryFirst,
	})
	updatedConfig := testAccIngestRouteConfig(namePrefix, ingestRouteConfigOptions{
		primaryPipeline:           "filter true | filter true",
		primaryEnabled:            false,
		ingestRouteOrderDirection: ingestRouteOrderSecondaryFirst,
	})
	var expectedOrderRouteIDs []string

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_ingest_route.primary", "type", "otellogs"),
					resource.TestCheckResourceAttr("observe_ingest_route.primary", "enabled", "true"),
					resource.TestCheckResourceAttrSet("observe_ingest_route.primary", "secondary_destination_id"),
					testCheckIngestRouteOrder("observe_ingest_route.primary", "observe_ingest_route.secondary"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_ingest_route.primary", "pipeline", "filter true | filter true"),
					resource.TestCheckResourceAttr("observe_ingest_route.primary", "enabled", "false"),
					resource.TestCheckResourceAttr("observe_ingest_route.primary", "secondary_destination_id", ""),
					testCheckIngestRouteOrder("observe_ingest_route.secondary", "observe_ingest_route.primary"),
				),
			},
			{
				ResourceName:      "observe_ingest_route.primary",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "observe_ingest_route.secondary",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:            "observe_ingest_route_order.otellogs",
				ImportState:             true,
				ImportStateIdFunc:       ingestRouteOrderImportID(&expectedOrderRouteIDs),
				ImportStateCheck:        checkImportedIngestRouteOrder(&expectedOrderRouteIDs),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"route_ids"},
			},
			{
				Config:   updatedConfig,
				PlanOnly: true,
			},
		},
	})
}

func ingestRouteOrderImportID(expectedRouteIDs *[]string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		first, second, err := ingestRouteStatePair(state, "observe_ingest_route.secondary", "observe_ingest_route.primary")
		if err != nil {
			return "", err
		}
		*expectedRouteIDs = []string{first.Primary.Attributes["route_id"], second.Primary.Attributes["route_id"]}

		order, ok := state.RootModule().Resources["observe_ingest_route_order.otellogs"]
		if !ok || order.Primary == nil {
			return "", fmt.Errorf("ingest route order resource not found in Terraform state")
		}
		return order.Primary.ID, nil
	}
}

func checkImportedIngestRouteOrder(expectedRouteIDs *[]string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(*expectedRouteIDs) != 2 {
			return fmt.Errorf("expected exactly two route IDs, got %d", len(*expectedRouteIDs))
		}
		if len(states) != 1 {
			return fmt.Errorf("imported %d resource states, want 1", len(states))
		}
		for index, expectedRouteID := range *expectedRouteIDs {
			key := fmt.Sprintf("route_ids.%d", index)
			if got := states[0].Attributes[key]; got != expectedRouteID {
				return fmt.Errorf("imported %s = %q, want %q", key, got, expectedRouteID)
			}
		}
		return nil
	}
}

func testCheckIngestRouteOrder(firstResourceName, secondResourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		first, second, err := ingestRouteStatePair(state, firstResourceName, secondResourceName)
		if err != nil {
			return err
		}
		order, ok := state.RootModule().Resources["observe_ingest_route_order.otellogs"]
		if !ok || order.Primary == nil {
			return fmt.Errorf("ingest route order resource not found in Terraform state")
		}
		if got, want := order.Primary.Attributes["route_ids.0"], first.Primary.Attributes["route_id"]; got != want {
			return fmt.Errorf("first route ID = %q, want %q", got, want)
		}
		if got, want := order.Primary.Attributes["route_ids.1"], second.Primary.Attributes["route_id"]; got != want {
			return fmt.Errorf("second route ID = %q, want %q", got, want)
		}
		return nil
	}
}

func ingestRouteStatePair(state *terraform.State, firstResourceName, secondResourceName string) (*terraform.ResourceState, *terraform.ResourceState, error) {
	first, ok := state.RootModule().Resources[firstResourceName]
	if !ok || first.Primary == nil {
		return nil, nil, fmt.Errorf("resource %q not found in Terraform state", firstResourceName)
	}
	second, ok := state.RootModule().Resources[secondResourceName]
	if !ok || second.Primary == nil {
		return nil, nil, fmt.Errorf("resource %q not found in Terraform state", secondResourceName)
	}
	return first, second, nil
}

type ingestRouteOrderDirection string

const (
	ingestRouteOrderPrimaryFirst   ingestRouteOrderDirection = "primary_first"
	ingestRouteOrderSecondaryFirst ingestRouteOrderDirection = "secondary_first"
)

type ingestRouteConfigOptions struct {
	primaryPipeline           string
	primaryEnabled            bool
	includePrimarySecondary   bool
	ingestRouteOrderDirection ingestRouteOrderDirection
}

func testAccIngestRouteConfig(namePrefix string, options ingestRouteConfigOptions) string {
	primarySecondaryConfig := ""
	if options.includePrimarySecondary {
		primarySecondaryConfig = "\n  secondary_destination_id = data.observe_oid.secondary_dataset.id"
	}
	primaryEnabledConfig := ""
	if !options.primaryEnabled {
		primaryEnabledConfig = "\n  enabled = false"
	}

	return fmt.Sprintf(configPreamble+`
resource "observe_datastream" "primary" {
  workspace = data.observe_workspace.default.oid
  name      = "%[1]s-primary"
  type      = "OtelLogs"
}

resource "observe_datastream" "secondary" {
  workspace = data.observe_workspace.default.oid
  name      = "%[1]s-secondary"
  type      = "OtelLogs"
}

data "observe_oid" "primary_dataset" {
  oid = observe_datastream.primary.dataset
}

data "observe_oid" "secondary_dataset" {
  oid = observe_datastream.secondary.dataset
}

resource "observe_ingest_route" "primary" {
  type           = "otellogs"
  pipeline       = "%[2]s"
  destination_id = data.observe_oid.primary_dataset.id%[3]s%[4]s
}

resource "observe_ingest_route" "secondary" {
  type           = "otellogs"
  pipeline       = "filter true | filter false"
  destination_id = data.observe_oid.secondary_dataset.id
}

resource "observe_ingest_route_order" "otellogs" {
  type      = "otellogs"
  route_ids = [%[5]s]
}
`, namePrefix, options.primaryPipeline, primarySecondaryConfig, primaryEnabledConfig, ingestRouteOrderConfig(options.ingestRouteOrderDirection))
}

func ingestRouteOrderConfig(direction ingestRouteOrderDirection) string {
	switch direction {
	case ingestRouteOrderPrimaryFirst:
		return "observe_ingest_route.primary.route_id, observe_ingest_route.secondary.route_id"
	case ingestRouteOrderSecondaryFirst:
		return "observe_ingest_route.secondary.route_id, observe_ingest_route.primary.route_id"
	default:
		panic(fmt.Sprintf("unsupported ingest route order direction %q", direction))
	}
}
