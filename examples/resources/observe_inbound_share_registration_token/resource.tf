# See observe_inbound_share for the full setup
resource "observe_inbound_share_registration_token" "acme" {
  provider_account = "ACME_CORP.PROD"
}

resource "snowflake_share" "customer_data" {
  name    = "CUSTOMER_SHARE_PROD"
  comment = observe_inbound_share_registration_token.acme.token
}
