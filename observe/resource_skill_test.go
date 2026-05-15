package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func TestAccObserveSkill(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_skill" "test" {
						label       = "%[1]s"
						description = "Test skill description"
						content     = "# Test Skill\n\nThis is a test skill."
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_skill.test", "label", randomPrefix),
					resource.TestCheckResourceAttr("observe_skill.test", "description", "Test skill description"),
					resource.TestCheckResourceAttr("observe_skill.test", "content", "# Test Skill\n\nThis is a test skill."),
					resource.TestCheckResourceAttr("observe_skill.test", "visibility", "Workspace"),
					resource.TestCheckResourceAttrSet("observe_skill.test", "oid"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "observe_skill" "test" {
						label       = "%[1]s-updated"
						description = "Updated description"
						content     = "# Updated Skill\n\nUpdated content."
						visibility  = "Private"
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_skill.test", "label", randomPrefix+"-updated"),
					resource.TestCheckResourceAttr("observe_skill.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("observe_skill.test", "content", "# Updated Skill\n\nUpdated content."),
					resource.TestCheckResourceAttr("observe_skill.test", "visibility", "Private"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "observe_skill" "test" {
						label       = "%[1]s-updated"
						description = "Updated description"
						content     = "# Updated Skill\n\nUpdated content."
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_skill.test", "visibility", "Workspace"),
				),
			},
		},
	})
}

func TestAccObserveSkillDataSource(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "observe_skill" "test" {
						label       = "%[1]s"
						description = "Test skill for data source"
						content     = "# Data Source Test"
					}

					data "observe_skill" "lookup" {
						id = observe_skill.test.id
					}
				`, randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_skill.lookup", "label", randomPrefix),
					resource.TestCheckResourceAttr("data.observe_skill.lookup", "description", "Test skill for data source"),
					resource.TestCheckResourceAttr("data.observe_skill.lookup", "content", "# Data Source Test"),
					resource.TestCheckResourceAttr("data.observe_skill.lookup", "visibility", "Workspace"),
					resource.TestCheckResourceAttrSet("data.observe_skill.lookup", "oid"),
				),
			},
		},
	})
}

func TestSkillVisibilityRoundTrip(t *testing.T) {
	for _, p := range []struct {
		tf  string
		api rest.SkillVisibility
	}{
		{"", rest.SkillVisibilityListed},
		{skillVisibilityWorkspace, rest.SkillVisibilityListed},
		{skillVisibilityPrivate, rest.SkillVisibilityUnlisted},
	} {
		api, err := skillAPIVisibilityFromTerraform(p.tf)
		if err != nil {
			t.Fatalf("skillAPIVisibilityFromTerraform(%q): %v", p.tf, err)
		}
		if api != p.api {
			t.Fatalf("skillAPIVisibilityFromTerraform(%q) = %q, want %q", p.tf, api, p.api)
		}
	}
	tf, err := skillTerraformVisibilityFromAPI(rest.SkillVisibilityListed)
	if err != nil || tf != skillVisibilityWorkspace {
		t.Fatalf("Listed -> Terraform: got %q, %v", tf, err)
	}
	tf, err = skillTerraformVisibilityFromAPI(rest.SkillVisibilityUnlisted)
	if err != nil || tf != skillVisibilityPrivate {
		t.Fatalf("Unlisted -> Terraform: got %q, %v", tf, err)
	}
}
