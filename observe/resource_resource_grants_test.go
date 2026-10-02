package observe

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The plan must decide replacement the same way before and after the target's
// oid becomes known, or Terraform fails with "inconsistent final plan" (OB-68049).
func TestResourceGrantsOIDChangeDiff(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	for _, testCase := range []struct {
		name, oldOID, newOID string
		wantDiff, wantNew    bool
	}{
		{name: "unversioned target replaced, unknown at plan", oldOID: "o:::dataset:41084454", newOID: unknown, wantDiff: true, wantNew: true},
		{name: "unversioned target replaced, known at apply", oldOID: "o:::dataset:41084454", newOID: "o:::dataset:41099999", wantDiff: true, wantNew: true},
		{name: "versioned target edited, unknown at plan", oldOID: "o:::dataset:41084454/1700000000", newOID: unknown, wantDiff: true},
		{name: "version only", oldOID: "o:::dataset:41084454/1700000000", newOID: "o:::dataset:41084454/1700000001"},
		{name: "version added", oldOID: "o:::dataset:41084454", newOID: "o:::dataset:41084454/1700000000"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := &terraform.InstanceState{ID: testCase.oldOID, Attributes: map[string]string{
				"id": testCase.oldOID, "oid": testCase.oldOID,
			}}
			config := terraform.NewResourceConfigRaw(map[string]interface{}{"oid": testCase.newOID})
			diff, err := resourceResourceGrants().Diff(context.Background(), state, config, nil)
			if err != nil {
				t.Fatal(err)
			}
			if gotDiff := diff != nil; gotDiff != testCase.wantDiff {
				t.Fatalf("diff = %v, want diff %v", diff, testCase.wantDiff)
			}
			if diff != nil && diff.RequiresNew() != testCase.wantNew {
				t.Errorf("RequiresNew = %v, want %v", diff.RequiresNew(), testCase.wantNew)
			}
		})
	}
}

func TestAccObserveResourceGrantsDataset(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-1"
					inputs = {
						"test" = observe_datastream.test.dataset
					}
					stage {}
				}

				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				data "observe_rbac_group" "everyone" {
					name = "Everyone"
				}

				resource "observe_resource_grants" "test" {
					oid = observe_dataset.test.oid

					grant {
						subject = observe_rbac_group.example.oid
						role    = "dataset_editor"
					}
					grant {
						subject = data.observe_rbac_group.everyone.oid
						role    = "dataset_viewer"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_resource_grants.test", "oid"),
					resource.TestCheckResourceAttr("observe_resource_grants.test", "grant.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_editor"}),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_viewer"}),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+datastreamConfigPreamble+`
				resource "observe_dataset" "test" {
					workspace = data.observe_workspace.default.oid
					name      = "%[1]s-1"
					inputs = {
						"test" = observe_datastream.test.dataset
					}
					stage {}
				}

				resource "observe_rbac_group" "example" {
					name      = "%[1]s"
				}

				data "observe_rbac_group" "everyone" {
					name = "Everyone"
				}

				resource "observe_resource_grants" "test" {
					oid = observe_dataset.test.oid

					grant {
						subject = observe_rbac_group.example.oid
						role    = "dataset_editor"
					}
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("observe_resource_grants.test", "oid"),
					resource.TestCheckResourceAttr("observe_resource_grants.test", "grant.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("observe_resource_grants.test", "grant.*", map[string]string{"role": "dataset_editor"}),
				),
			},
		},
	})
}
