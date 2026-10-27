data "observe_inbound_share_setup" "setup" {}

# Only needed for the first share registered from a provider account
resource "observe_inbound_share_registration_token" "acme" {
  provider_account = "ACME_CORP.PROD"
}

# Grant the share to Observe, and prove ownership by setting its comment to
# the token
resource "snowflake_share" "customer_data" {
  name     = "CUSTOMER_SHARE_PROD"
  comment  = observe_inbound_share_registration_token.acme.token
  accounts = [data.observe_inbound_share_setup.setup.consumer_account]
}

resource "observe_inbound_share" "customer_data" {
  share_name       = snowflake_share.customer_data.name
  provider_account = observe_inbound_share_registration_token.acme.provider_account
  label            = "Customer data"
}

resource "observe_inbound_share_table" "events" {
  share_id         = observe_inbound_share.customer_data.oid
  table_name       = "EVENTS"
  schema_name      = "PUBLIC"
  dataset_label    = "Customer Events"
  dataset_kind     = "Event"
  valid_from_field = "EVENT_TIMESTAMP"
}
