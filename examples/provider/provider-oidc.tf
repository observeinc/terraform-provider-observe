provider "observe" {
  customer = 123456789012
  domain   = "observeinc.com"

  # Requires configuring the OIDC integration in Observe first.
  # The provider will automatically fetch the OIDC token from the environment
  # (e.g., GitHub Actions or Terraform Cloud).
  oauth2 {
    client_id = "my-client-id"
    token_url = "https://login.microsoftonline.com/<tenant>/oauth2/v2.0/token"
  }
}
