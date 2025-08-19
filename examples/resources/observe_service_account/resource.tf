# Create a service account for automated API access
resource "observe_service_account" "automation" {
  label       = "automation-service-account"
  description = "Service account for automated api access"
  disabled    = false
}

# Create a service account for CI/CD pipeline
resource "observe_service_account" "cicd" {
  label       = "cicd-pipeline"
  description = "Service account for CI/CD pipeline integration"
  disabled    = false
}

# Create a disabled service account (for testing or temporary use)
resource "observe_service_account" "test" {
  label       = "test-service-account"
  description = "Test service account - disabled by default"
  disabled    = true
}

# Output service account information
output "automation_info" {
  value = {
    label       = observe_service_account.automation.label
    description = observe_service_account.automation.description
    disabled    = observe_service_account.automation.disabled
  }
  description = "Information about the automation service account"
}
