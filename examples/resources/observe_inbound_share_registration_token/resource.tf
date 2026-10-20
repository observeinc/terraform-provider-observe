resource "observe_inbound_share_registration_token" "acme" {
  provider_account = "ACME_CORP.PROD"
}

# Prove ownership by setting the share comment to the token
resource "snowflake_share" "customer_data" {
  name     = "CUSTOMER_SHARE_PROD"
  comment  = observe_inbound_share_registration_token.acme.token
  accounts = [data.observe_inbound_share_setup.setup.consumer_account]
}

data "observe_inbound_share_setup" "setup" {}
