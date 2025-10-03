# Create a service account
resource "observe_service_account" "automation" {
  label       = "automation-service-account"
  description = "Service account for automated API access"
  disabled    = false
}

# Create an API token for the service account with 30-day expiration
resource "observe_service_account_token" "automation_token" {
  service_account = observe_service_account.automation.oid
  label           = "automation-token"
  description     = "API token for automation scripts"
  expiration      = "2030-01-01T00:00:00Z"
}

# Create a short-lived API token for testing (1 day)
resource "observe_service_account_token" "test_token" {
  service_account = observe_service_account.automation.oid
  label           = "test-token"
  description     = "Short-lived token for testing"
  expiration      = "2030-01-02T00:00:00Z"
}

# Create a disabled API token (can be enabled later)
resource "observe_service_account_token" "disabled_token" {
  service_account = observe_service_account.automation.oid
  label           = "disabled-token"
  description     = "Disabled token for future use"
  expiration      = "2030-01-07T00:00:00Z"
  disabled        = true
}

# Output the secret (only available on creation)
output "automation_token_secret" {
  value     = observe_service_account_api_token.automation_token.secret
  sensitive = true
}

# Output the expiration time
output "automation_token_expiration" {
  value = observe_service_account_api_token.automation_token.expiration
}

