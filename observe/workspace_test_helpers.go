package observe

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	defaultWorkspaceName = getenv("OBSERVE_WORKSPACE", "Default")
	configPreamble       = fmt.Sprintf(`
				data "observe_workspace" "default" {
					name = "%s"
				}`, defaultWorkspaceName)

	// datastreamNoWorkspacePreamble creates a datastream without specifying workspace.
	datastreamNoWorkspacePreamble = `
	resource "observe_datastream" "test_no_ws" {
		name = "%[1]s"
	}`
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func datastreamNoWorkspacePreambleFmt(prefix string) string {
	return fmt.Sprintf(datastreamNoWorkspacePreamble, prefix)
}

// linkConfigNoWorkspacePreamble sets up two linked datasets without workspace on any resource.
func linkConfigNoWorkspacePreamble(prefix string) string {
	return datastreamNoWorkspacePreambleFmt(prefix) + fmt.Sprintf(`
		resource "observe_dataset" "a" {
			name = "%[1]s-A"

			inputs = { "test" = observe_datastream.test_no_ws.dataset }

			stage {
				pipeline = <<-EOF
					filter false
					colmake key:"test"
				EOF
			}
		}

		resource "observe_dataset" "b" {
			name = "%[1]s-B"

			inputs = { "a" = observe_dataset.a.oid }

			stage {
				pipeline = <<-EOF
					makeresource primarykey(key)
				EOF
			}
		}`, prefix)
}

func testAccPlanOnlyNoDriftStep(config string) resource.TestStep {
	return resource.TestStep{
		Config:             config,
		PlanOnly:           true,
		ExpectNonEmptyPlan: false,
	}
}

func testAccNoWorkspaceSteps(config string, checks ...resource.TestCheckFunc) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config,
			Check:  resource.ComposeTestCheckFunc(checks...),
		},
		testAccPlanOnlyNoDriftStep(config),
	}
}
