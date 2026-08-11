package observe

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Dataset edits must not recreate the tag, and the plan must decide replacement the
// same way before and after the dataset oid becomes known, or Terraform fails with
// "inconsistent final plan".
func TestCorrelationTagDatasetChangeDiff(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	for _, testCase := range []struct {
		name, oldOID, newOID string
		wantDiff, wantNew    bool
	}{
		{name: "version only", oldOID: "o:::dataset:41084454/1700000000", newOID: "o:::dataset:41084454/1700000001"},
		{name: "versioned dataset edited, unknown at plan", oldOID: "o:::dataset:41084454/1700000000", newOID: unknown, wantDiff: true},
		{name: "unversioned dataset replaced, unknown at plan", oldOID: "o:::dataset:41084454", newOID: unknown, wantDiff: true, wantNew: true},
		{name: "unversioned dataset replaced, known at apply", oldOID: "o:::dataset:41084454", newOID: "o:::dataset:41099999", wantDiff: true, wantNew: true},
		{name: "different dataset", oldOID: "o:::dataset:41084454/1700000000", newOID: "o:::dataset:41099999/1700000000", wantDiff: true, wantNew: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := &terraform.InstanceState{ID: "tag", Attributes: map[string]string{
				"id": "tag", "dataset": testCase.oldOID, "name": "tag", "column": "key",
			}}
			config := terraform.NewResourceConfigRaw(map[string]interface{}{
				"dataset": testCase.newOID, "name": "tag", "column": "key",
			})
			diff, err := resourceCorrelationTag().Diff(context.Background(), state, config, nil)
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

func TestCorrelationTagCreation(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", fmt.Sprintf("%s-key.name", randomPrefix)),
					resource.TestCheckResourceAttr("observe_correlation_tag.example", "column", "key"),
					resource.TestCheckResourceAttrSet("observe_correlation_tag.example", "dataset"),
				),
			},
			// Using the same config, there should not be any diff.
			{
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Making any change to the config should delete and recreate the tag (in-place update is not supported)
				Config: fmt.Sprintf(linkConfigPreamble+`
					resource "observe_correlation_tag" "example" {
					name = "%[1]s-key.name-2"
					dataset = observe_dataset.a.oid
					column = "key"
				}`, randomPrefix),
				Check: resource.TestCheckResourceAttr("observe_correlation_tag.example", "name", fmt.Sprintf("%s-key.name-2", randomPrefix)),
			},
		},
	})
}
