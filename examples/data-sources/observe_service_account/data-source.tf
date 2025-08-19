# Look up a service account by ID
data "observe_service_account" "automation" {
  id = "12345678"
}

# Use the service account data in other resources
output "service_account_info" {
  value = {
    label       = data.observe_service_account.automation.label
    disabled    = data.observe_service_account.automation.disabled
    description = data.observe_service_account.automation.description
  }
}
