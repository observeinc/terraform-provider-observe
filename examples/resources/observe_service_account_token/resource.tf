# Create a service account
resource "observe_service_account" "automation" {
  label       = "automation-service-account"
  description = "Service account for automated API access"
  disabled    = false
}

# Create a short-lived API token for the service account with 1-hour lifetime
resource "observe_service_account_token" "short_lived" {
  service_account = observe_service_account.automation.oid
  label           = "short-lived-token"
  description     = "Temporary API token"
  lifetime_hours  = 1
  disabled        = false
}

# Create a long-lived API token (365 days)
resource "observe_service_account_token" "long_lived" {
  service_account = observe_service_account.automation.oid
  label           = "long-lived-token"
  description     = "API token for long-running integrations"
  lifetime_hours  = 8760 # 365 days
  disabled        = false
}
