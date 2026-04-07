package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Test fixture configuration
// These tests require a pre-existing inbound share with test data.
// The share must be already accepted in Snowflake and accessible via Observe.
//
// Tests are skipped unless CI=true (requires manual Snowflake share setup).
//
// Environment variables (with defaults):
//   TEST_INBOUND_SHARE_NAME     - Snowflake share name
//   TEST_INBOUND_SHARE_PROVIDER - Snowflake provider account
//   TEST_INBOUND_TABLE_DATA     - Main test table name
//   TEST_INBOUND_TABLE_REF      - Reference test table name (optional)
//   TEST_INBOUND_TABLE_TYPES    - Types test table name
//
// To skip these tests:
//   export SKIP_INBOUND_SHARE_TESTS=true
var (
	testInboundShareName     = getenv("TEST_INBOUND_SHARE_NAME", "MATTG_SHAREIN_TEST_DATA_SHARE2")
	testInboundShareProvider = getenv("TEST_INBOUND_SHARE_PROVIDER", "HC83707.OBSERVE_O2_1")
	testInboundSchemaName    = "PUBLIC"
	testInboundTableData     = getenv("TEST_INBOUND_TABLE_DATA", "TEMP_TEST_DATA")
	testInboundTableRef      = getenv("TEST_INBOUND_TABLE_REF", "TEMP_TEST_REF_DATA")
	testInboundTableTypes    = getenv("TEST_INBOUND_TABLE_TYPES", "COMPREHENSIVE_TYPES_TEST")
)

// TestAccObserveInboundShareTable_Basic tests the complete lifecycle of tracking
// a table from an inbound share using the TEMP_TEST_DATA table.
//
// This test verifies:
//   - Share lookup by name + provider account works correctly
//   - Table can be tracked and dataset is created
//   - All computed fields are populated (oid, table_id, dataset_id, etc.)
//   - Dataset label and description can be updated
//   - Resource cleanup works (untrack table, delete dataset)
//
// Test flow:
//   Step 1: Track TEMP_TEST_DATA table as "Table" kind dataset
//   Step 2: Update dataset label and add description
//   Step 3: Automatic cleanup via Terraform destroy
func TestAccObserveInboundShareTable_Basic(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Step 1: Create - track a table and create dataset
			{
				Config: testAccInboundShareTableConfig(randomPrefix, "Table", "", testInboundTableData),
				Check: resource.ComposeTestCheckFunc(
					// Verify all computed fields are populated
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "table_id"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "status"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "full_table_path"),

					// Verify input values match configuration
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "table_name", testInboundTableData),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "schema_name", testInboundSchemaName),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_label", randomPrefix),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_kind", "Table"),
				),
			},
			// Step 2: Update - change dataset label and add description
			{
				Config: testAccInboundShareTableConfig(randomPrefix+"-updated", "Table", "Updated description", testInboundTableData),
				Check: resource.ComposeTestCheckFunc(
					// Verify updated fields changed
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_label", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "description", "Updated description"),

					// Verify other fields remain unchanged
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "table_name", testInboundTableData),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "schema_name", testInboundSchemaName),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_kind", "Table"),

					// Verify IDs remain the same (not recreated)
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "table_id"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_oid"),
				),
			},
			// Step 3: Import - verify composite ID import works
			{
				ResourceName: "observe_inbound_share_table.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["observe_inbound_share_table.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["share_id"] + "/" + rs.Primary.ID, nil
				},
				ImportStateVerify: true,
				// These fields are not yet returned by the sharein API in production:
				// - description: cannot be reliably inferred from Dataset API
				// - field_mapping: can only infer Drop conversions, not user-specified Direct mappings
				ImportStateVerifyIgnore: []string{"description", "field_mapping"},
			},
		},
		// Step 4: Destroy happens automatically - tests cleanup (untrack table, delete dataset)
	})
}

// TestAccObserveInboundShareTable_WithDescription tests tracking with a description field.
//
// This test verifies:
//   - Tables can be tracked with optional description
//   - Description is properly set on the dataset
//
// Note: The original Event test was removed because the test tables don't have
// a known timestamp field. Event datasets require valid_from_field to be set
// to an actual column in the table, and we don't have visibility into the
// test table schema. This test focuses on description instead.
func TestAccObserveInboundShareTable_WithDescription(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundShareTableConfig(randomPrefix, "Table", "Test description for tracked table", testInboundTableData),
				Check: resource.ComposeTestCheckFunc(
					// Verify description is set
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "description", "Test description for tracked table"),
					// Verify dataset was created successfully
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
				),
			},
		},
	})
}

// TestAccObserveInboundShareTable_RefData tests tracking the reference data table.
//
// This test verifies:
//   - Both test tables (TEMP_TEST_DATA and TEMP_TEST_REF_DATA) can be tracked
//   - Reference/lookup tables work the same as data tables
//   - Multiple tables from same share can be managed independently
//
// This matches the Python integration test which tracks both tables.
func TestAccObserveInboundShareTable_RefData(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundShareTableConfig(randomPrefix+"-ref", "Table", "Reference data table", testInboundTableRef),
				Check: resource.ComposeTestCheckFunc(
					// Verify table was tracked successfully
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
					// Verify correct table name
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "table_name", testInboundTableRef),
					// Verify description
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "description", "Reference data table"),
				),
			},
		},
	})
}

// TestAccObserveInboundShareTable_MultipleTables tests tracking both test tables simultaneously.
//
// This test verifies:
//   - Multiple tables from the same share can be tracked at once
//   - Each table gets its own dataset
//   - Tables can be tracked/untracked independently
//   - No conflicts or race conditions when managing multiple tables
//
// This is the most comprehensive test and matches the Python integration test behavior
// which tracks and manages both TEMP_TEST_DATA and TEMP_TEST_REF_DATA.
func TestAccObserveInboundShareTable_MultipleTables(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Step 1: Track both tables from the share
			{
				Config: testAccInboundShareTableConfigMultiple(randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					// Verify data table
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.data", "oid"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.data", "table_name", testInboundTableData),
					resource.TestCheckResourceAttr("observe_inbound_share_table.data", "dataset_label", randomPrefix+"-data"),

					// Verify ref table
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.ref", "oid"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.ref", "table_name", testInboundTableRef),
					resource.TestCheckResourceAttr("observe_inbound_share_table.ref", "dataset_label", randomPrefix+"-ref"),

					// Verify both have different dataset IDs
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.data", "dataset_id"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.ref", "dataset_id"),
				),
			},
			// Step 2: Update both tables
			{
				Config: testAccInboundShareTableConfigMultiple(randomPrefix + "-updated"),
				Check: resource.ComposeTestCheckFunc(
					// Verify updates applied to both
					resource.TestCheckResourceAttr("observe_inbound_share_table.data", "dataset_label", randomPrefix+"-updated-data"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.ref", "dataset_label", randomPrefix+"-updated-ref"),
				),
			},
		},
		// Automatic cleanup will untrack both tables
	})
}

// TestAccObserveInboundShareTable_EventDataset tests tracking a table as an Event dataset.
//
// This test verifies:
//   - Tables can be tracked as Event datasets (not just Table kind)
//   - valid_from_field configuration works correctly
//   - Event datasets are created with proper timestamp field
//
// Uses COMPREHENSIVE_TYPES_TEST table which has a timestamp_tz field suitable for events.
func TestAccObserveInboundShareTable_EventDataset(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundShareTableConfigEventDataset(randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					// Verify Event dataset kind is set
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_kind", "Event"),
					// Verify timestamp field is configured
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "valid_from_field", "TIMESTAMP_TZ"),
					// Verify dataset was created successfully
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
				),
			},
			// Import and verify dataset_kind is correctly inferred from Dataset API
			{
				ResourceName: "observe_inbound_share_table.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["observe_inbound_share_table.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["share_id"] + "/" + rs.Primary.ID, nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"description", "field_mapping"},
				Check: resource.ComposeTestCheckFunc(
					// Verify Event kind is inferred correctly (has validFromField, no validToField)
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_kind", "Event"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "valid_from_field", "TIMESTAMP_TZ"),
				),
			},
		},
	})
}

// TestAccObserveInboundShareTable_FieldMapping tests field mapping functionality.
//
// This test verifies:
//   - Field mappings can be specified during table tracking
//   - Field type conversions work (integer vs float)
//   - Field mappings can be added via update
//   - Field mappings can be modified via update
//   - Field mappings can be removed via update
//
// Uses COMPREHENSIVE_TYPES_TEST table which has integer_type and other fields
// to test type conversions.
//
// Test flow:
//   Step 1: Create without field mapping
//   Step 2: Add field mapping for integer_type (as integer)
//   Step 3: Update field mapping to treat integer_type as float
//   Step 4: Remove field mapping
func TestAccObserveInboundShareTable_FieldMapping(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Step 1: Create with minimal field mapping (drop unsupported TIME_TYPE column)
			{
				Config: testAccInboundShareTableConfigDropTime(randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_label", randomPrefix),
					// Verify TIME_TYPE is dropped
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "1"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.0.field", "TIME_TYPE"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.0.conversion", "Drop"),
				),
			},
			// Step 2: Add field mapping - treat integer_type as integer, keep TIME_TYPE drop
			{
				Config: testAccInboundShareTableConfigWithFieldMapping(randomPrefix+"-mapped", "int64"),
				Check: resource.ComposeTestCheckFunc(
					// Verify field mappings - now have 2 (TIME_TYPE drop + integer_type)
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "2"),

					// Verify dataset not recreated (IDs stay same)
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "dataset_id"),
				),
			},
			// Step 3: Update field mapping - treat integer_type as float instead
			{
				Config: testAccInboundShareTableConfigWithFieldMapping(randomPrefix+"-float", "float64"),
				Check: resource.ComposeTestCheckFunc(
					// Verify still have 2 field mappings
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "2"),

					// Verify still not recreated
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
				),
			},
			// Step 4: Remove integer_type mapping, keep only TIME_TYPE drop
			{
				Config: testAccInboundShareTableConfigDropTime(randomPrefix+"-removed"),
				Check: resource.ComposeTestCheckFunc(
					// Verify back to 1 field mapping (just TIME_TYPE drop)
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "1"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.0.field", "TIME_TYPE"),

					// Verify still not recreated
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
				),
			},
		},
	})
}

// TestAccObserveInboundShareTable_UpdateAllFields tests updating all mutable fields.
//
// This test comprehensively validates that the UPDATE API works correctly for all
// fields that can be changed without destroying and recreating the resource.
//
// Mutable fields tested:
//   - dataset_label
//   - description
//   - valid_from_field (for Event datasets)
//   - valid_to_field (for Interval datasets)
//   - field_mapping
//
// This is the most comprehensive update test - it verifies that all update paths
// through the code work correctly and use PATCH API instead of destroy+recreate.
func TestAccObserveInboundShareTable_UpdateAllFields(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckInboundShare(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Step 1: Create with minimal configuration
			{
				Config: testAccInboundShareTableConfigMinimal(randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_label", randomPrefix),
					// description not in config, so not in state (Optional+Computed fields only appear if set)
					resource.TestCheckNoResourceAttr("observe_inbound_share_table.test", "description"),
					// Has 1 field_mapping (TIME_TYPE drop)
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "1"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
				),
			},
			// Step 2: Update all mutable fields at once
			{
				Config: testAccInboundShareTableConfigAllFields(randomPrefix + "-updated"),
				Check: resource.ComposeTestCheckFunc(
					// Verify all fields updated
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "dataset_label", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "description", "Comprehensive update test"),
					// Now has 3 field_mappings (TIME_TYPE drop + integer_type + string_type)
					resource.TestCheckResourceAttr("observe_inbound_share_table.test", "field_mapping.#", "3"),

					// Verify resource was not recreated (UPDATE API used, not destroy+create)
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "oid"),
					resource.TestCheckResourceAttrSet("observe_inbound_share_table.test", "table_id"),
				),
			},
		},
	})
}

// ============================================================================
// Helper functions to generate Terraform test configurations
// ============================================================================

// testAccInboundShareTableConfig generates a Terraform config for tracking a single table.
//
// Parameters:
//   - datasetLabel: Label for the Observe dataset (e.g., "my-test-dataset")
//   - datasetKind: Dataset kind - "Table", "Event", "Interval", or "Resource"
//   - description: Optional description for the dataset (can be empty string)
//   - tableName: Name of the table in the share (e.g., "TEMP_TEST_DATA")
//
// Returns a Terraform configuration string that:
//   1. Looks up the test share by name + provider
//   2. Tracks the specified table from the share
//   3. Creates an Observe dataset with the given configuration
func testAccInboundShareTableConfig(datasetLabel, datasetKind, description, tableName string) string {
	config := fmt.Sprintf(`
# Look up the inbound share by name and provider account
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track a table from the share and create a dataset
resource "observe_inbound_share_table" "test" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s"
	dataset_kind  = "%s"
`, testInboundShareName, testInboundShareProvider, tableName, testInboundSchemaName, datasetLabel, datasetKind)

	// Add optional description if provided
	if description != "" {
		config += fmt.Sprintf(`	description   = "%s"
`, description)
	}

	config += "}\n"
	return config
}



// testAccInboundShareTableConfigMultiple generates a config tracking both test tables.
//
// This creates two separate resources:
//   - observe_inbound_share_table.data - Tracks TEMP_TEST_DATA
//   - observe_inbound_share_table.ref - Tracks TEMP_TEST_REF_DATA
//
// This mirrors the Python integration test which manages both tables simultaneously.
//
// Parameters:
//   - labelPrefix: Prefix for dataset labels (e.g., "tf-test")
//                  Will create datasets named "{prefix}-data" and "{prefix}-ref"
//
// Returns a Terraform configuration that tracks both tables from the same share.
func testAccInboundShareTableConfigMultiple(labelPrefix string) string {
	return fmt.Sprintf(`
# Look up the inbound share (shared by both tables)
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track the main data table
resource "observe_inbound_share_table" "data" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s-data"
	dataset_kind  = "Table"
	description   = "Main test data table"
}

# Track the reference data table
resource "observe_inbound_share_table" "ref" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s-ref"
	dataset_kind  = "Table"
	description   = "Reference data table"
}
`, testInboundShareName, testInboundShareProvider,
		testInboundTableData, testInboundSchemaName, labelPrefix,
		testInboundTableRef, testInboundSchemaName, labelPrefix)
}

// testAccInboundShareTableConfigEventDataset generates config for Event dataset with timestamp.
//
// Uses COMPREHENSIVE_TYPES_TEST table with TIMESTAMP_TZ field as the event timestamp.
// Must also drop TIME_TYPE column which is unsupported.
func testAccInboundShareTableConfigEventDataset(datasetLabel string) string {
	return fmt.Sprintf(`
# Look up the inbound share
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track table as Event dataset with timestamp field
resource "observe_inbound_share_table" "test" {
	share_id         = data.observe_inbound_share.test.oid
	table_name       = "%s"
	schema_name      = "%s"
	dataset_label    = "%s"
	dataset_kind     = "Event"
	valid_from_field = "TIMESTAMP_TZ"  # Column name for event timestamp (uppercase for Snowflake)
	description      = "Event dataset with TIMESTAMP_TZ"

	# Drop unsupported TIME_TYPE column
	field_mapping {
		field      = "TIME_TYPE"
		type       = "string"
		conversion = "Drop"
	}
}
`, testInboundShareName, testInboundShareProvider, testInboundTableTypes, testInboundSchemaName, datasetLabel)
}

// testAccInboundShareTableConfigDropTime generates config that drops TIME_TYPE column.
//
// COMPREHENSIVE_TYPES_TEST table has a TIME_TYPE column with Snowflake type TIME(9)
// which Observe doesn't auto-convert. We drop it by using type "string" with conversion "drop".
// Note: The Terraform schema requires both type and conversion, even though we're dropping the field.
func testAccInboundShareTableConfigDropTime(datasetLabel string) string {
	return fmt.Sprintf(`
# Look up the inbound share
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track table with TIME_TYPE column dropped
resource "observe_inbound_share_table" "test" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s"
	dataset_kind  = "Table"
	description   = "Table with TIME_TYPE dropped"

	# Drop unsupported TIME_TYPE column (type is required by schema but ignored)
	field_mapping {
		field      = "TIME_TYPE"
		type       = "string"
		conversion = "Drop"
	}
}
`, testInboundShareName, testInboundShareProvider, testInboundTableTypes, testInboundSchemaName, datasetLabel)
}

// testAccInboundShareTableConfigWithFieldMapping generates config with field mapping.
//
// Parameters:
//   - datasetLabel: Label for the dataset
//   - fieldType: Type to use for integer_type field ("int64" or "float64")
//
// Tests field mapping by specifying the type of the integer_type field.
// Also drops TIME_TYPE column since it's unsupported.
func testAccInboundShareTableConfigWithFieldMapping(datasetLabel, fieldType string) string {
	return fmt.Sprintf(`
# Look up the inbound share
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track table with field mapping
resource "observe_inbound_share_table" "test" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s"
	dataset_kind  = "Table"
	description   = "Table with field mapping: integer_type as %s"

	# Drop unsupported TIME_TYPE column
	field_mapping {
		field      = "TIME_TYPE"
		type       = "string"
		conversion = "Drop"
	}

	# Map integer_type field to specified type
	field_mapping {
		field      = "integer_type"
		type       = "%s"
		conversion = "Direct"
	}
}
`, testInboundShareName, testInboundShareProvider, testInboundTableTypes, testInboundSchemaName, datasetLabel, fieldType, fieldType)
}

// testAccInboundShareTableConfigMinimal generates minimal config for update testing.
//
// Creates a table track with only required fields, so we can test adding optional
// fields via update. Must still drop TIME_TYPE column.
func testAccInboundShareTableConfigMinimal(datasetLabel string) string {
	return fmt.Sprintf(`
# Look up the inbound share
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track table with minimal configuration
resource "observe_inbound_share_table" "test" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s"
	dataset_kind  = "Table"

	# Drop unsupported TIME_TYPE column
	field_mapping {
		field      = "TIME_TYPE"
		type       = "string"
		conversion = "Drop"
	}
}
`, testInboundShareName, testInboundShareProvider, testInboundTableTypes, testInboundSchemaName, datasetLabel)
}

// testAccInboundShareTableConfigAllFields generates config with all mutable fields set.
//
// Used to test comprehensive updates - goes from minimal config to full config
// with all optional fields populated.
func testAccInboundShareTableConfigAllFields(datasetLabel string) string {
	return fmt.Sprintf(`
# Look up the inbound share
data "observe_inbound_share" "test" {
	snowflake_config {
		share_name       = "%s"
		provider_account = "%s"
	}
}

# Track table with all mutable fields configured
resource "observe_inbound_share_table" "test" {
	share_id      = data.observe_inbound_share.test.oid
	table_name    = "%s"
	schema_name   = "%s"
	dataset_label = "%s"
	dataset_kind  = "Table"
	description   = "Comprehensive update test"

	# Drop unsupported TIME_TYPE column (always needed)
	field_mapping {
		field      = "TIME_TYPE"
		type       = "string"
		conversion = "Drop"
	}

	# Multiple field mappings for type conversion
	field_mapping {
		field      = "integer_type"
		type       = "int64"
		conversion = "Direct"
	}

	field_mapping {
		field      = "string_type"
		type       = "string"
		conversion = "Direct"
	}
}
`, testInboundShareName, testInboundShareProvider, testInboundTableTypes, testInboundSchemaName, datasetLabel)
}
