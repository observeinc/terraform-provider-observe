provider "observe" {
  customer = 123456789012
  domain   = "observeinc.com"

  # Requires configuring the M2M OAuth integration in Observe first
  oauth2 {
    client_id     = "my-client-id"
    client_secret = "my-client-secret"
    token_url     = "https://login.microsoftonline.com/<tenant>/oauth2/v2.0/token"
    scope         = "api://<app-id>/.default"
  }
}
