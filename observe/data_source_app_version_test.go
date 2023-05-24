package observe

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObserveDataAppVersion_Simple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_app_version" "jenkins_version" {
				  module_id          = "observeinc/jenkins/observe"
				  version_constraint = "> 0.2.0, < 0.4.0"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_app_version.jenkins_version", "module_id", "observeinc/jenkins/observe"),
					resource.TestCheckResourceAttr("data.observe_app_version.jenkins_version", "version", "0.3.1"),
				),
			},
		},
	})
}

func TestAccObserveDataAppVersion_Prerelease(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_app_version" "jenkins_version" {
				  module_id          = "observeinc/jenkins/observe"
				  version_constraint = "~> 0.2.1-1.beta"
                  include_prerelease = true
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.observe_app_version.jenkins_version", "module_id", "observeinc/jenkins/observe"),
					resource.TestCheckResourceAttr("data.observe_app_version.jenkins_version", "version", "0.2.1-10.beta+ga9fa278"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_app_version" "jenkins_version" {
				  module_id          = "observeinc/jenkins/observe"
				  version_constraint = "~> 0.2.1-1.beta"
				}`),
				ExpectError: regexp.MustCompile("no matching version found"),
			},
		},
	})
}

func TestAccObserveDataAppVersion_BadConstraints(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble + `
				data "observe_app_version" "jenkins_version" {
				  module_id          = "observeinc/jenkins/observe"
				  version_constraint = "< 0.2.0, > 0.4.0"
				}`),
				ExpectError: regexp.MustCompile("no matching version found"),
			},
		},
	})
}
