package observe

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testAccPreCheckSourceDatasetIceberg skips unless a storage integration is
// available, since CI tenants may not have one.
func testAccPreCheckSourceDatasetIceberg(t *testing.T) string {
	// Called eagerly (before resource.Test), so skip like resource.Test would.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless env %q is set", resource.EnvTfAcc)
	}
	testAccPreCheck(t)
	si := os.Getenv("OBSERVE_TEST_STORAGE_INTEGRATION_OID")
	if si == "" {
		t.Skip("OBSERVE_TEST_STORAGE_INTEGRATION_OID not set")
	}
	return si
}

func TestAccObserveSourceDatasetNative(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "observe_source_dataset" "native" {
  label               = "%[1]s/Logs"
  deletion_protection = false
}`, prefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_source_dataset.native", "label", prefix+"/Logs"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "storage", "Native"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "schema_type", "Any"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "external_write", "false"),
					resource.TestCheckResourceAttrSet("observe_source_dataset.native", "oid"),
					resource.TestCheckResourceAttrSet("observe_source_dataset.native", "effective_data_retention_days"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "observe_source_dataset" "native" {
  label               = "%[1]s/Renamed"
  description         = "managed by terraform"
  data_retention_days = 7
  deletion_protection = false
  object_tags = {
    team = "a,b"
  }
}`, prefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_source_dataset.native", "label", prefix+"/Renamed"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "description", "managed by terraform"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "data_retention_days", "7"),
					resource.TestCheckResourceAttr("observe_source_dataset.native", "object_tags.team", "a,b"),
				),
			},
			{
				ResourceName:            "observe_source_dataset.native",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy", "deletion_protection"},
			},
		},
	})
}

func TestAccObserveSourceDatasetIceberg(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf")
	// Resolved before Steps: step configs are built before PreCheck runs.
	si := testAccPreCheckSourceDatasetIceberg(t)
	var firstID string
	config := func(storage string) string {
		integration := ""
		if storage == "Iceberg" {
			integration = fmt.Sprintf("storage_integration = %q", si)
		}
		return fmt.Sprintf(`
resource "observe_source_dataset" "iceberg" {
  label               = "%[1]s/Iceberg"
  storage             = %[2]q
  deletion_protection = false
  %[3]s
}`, prefix, storage, integration)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config("Iceberg"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_source_dataset.iceberg", "storage", "Iceberg"),
					resource.TestCheckResourceAttrSet("observe_source_dataset.iceberg", "storage_integration"),
					func(s *terraform.State) error {
						firstID = s.RootModule().Resources["observe_source_dataset.iceberg"].Primary.ID
						return nil
					},
				),
			},
			{
				ResourceName:            "observe_source_dataset.iceberg",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy", "deletion_protection"},
			},
			{
				// The SDK in use has no plancheck package; a changed ID shows
				// that changing a ForceNew storage attribute replaced the dataset.
				Config: config("Native"),
				Check: func(s *terraform.State) error {
					if id := s.RootModule().Resources["observe_source_dataset.iceberg"].Primary.ID; id == firstID {
						return fmt.Errorf("expected replacement, id unchanged (%s)", id)
					}
					return nil
				},
			},
		},
	})
}

func TestAccObserveSourceDatasetExternalWrite(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf")
	// Resolved before Steps: step configs are built before PreCheck runs.
	si := testAccPreCheckSourceDatasetIceberg(t)
	if os.Getenv("OBSERVE_TEST_ICEBERG_EXTERNAL_WRITE") != "true" {
		t.Skip("OBSERVE_TEST_ICEBERG_EXTERNAL_WRITE != true")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "observe_source_dataset" "ew" {
  label               = "%[1]s/External"
  storage             = "Iceberg"
  storage_integration = %[2]q
  external_write      = true
  deletion_protection = false
}`, prefix, si),
				Check: resource.TestCheckResourceAttr("observe_source_dataset.ew", "external_write", "true"),
			},
		},
	})
}
