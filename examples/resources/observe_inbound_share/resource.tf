locals {
  provider_account = "ACME_CORP.PROD"
}

data "observe_inbound_share_setup" "setup" {}

# Only needed for the first share registered from a provider account. Once
# that share is registered, remove this resource and the share comment.
resource "observe_inbound_share_registration_token" "acme" {
  provider_account = local.provider_account
}

# The share, in the provider account, managed with the Snowflake provider
# (snowflake_share needs "snowflake_share_resource" in its
# preview_features_enabled). The comment proves ownership.
resource "snowflake_share" "customer_data" {
  name     = "CUSTOMER_SHARE_PROD"
  comment  = observe_inbound_share_registration_token.acme.token
  accounts = [data.observe_inbound_share_setup.setup.consumer_account]
}

resource "snowflake_grant_privileges_to_share" "database" {
  to_share    = snowflake_share.customer_data.name
  privileges  = ["USAGE"]
  on_database = "CUSTOMER_DB"
}

resource "snowflake_grant_privileges_to_share" "schema" {
  to_share   = snowflake_share.customer_data.name
  privileges = ["USAGE"]
  on_schema  = "CUSTOMER_DB.PUBLIC"
}

resource "snowflake_grant_privileges_to_share" "events" {
  to_share   = snowflake_share.customer_data.name
  privileges = ["SELECT"]
  on_table   = "CUSTOMER_DB.PUBLIC.EVENTS"
}

resource "observe_inbound_share" "customer_data" {
  share_name       = snowflake_share.customer_data.name
  provider_account = local.provider_account
  label            = "Customer data"

  depends_on = [snowflake_grant_privileges_to_share.database]
}

resource "observe_inbound_share_table" "events" {
  share_id         = observe_inbound_share.customer_data.oid
  schema_name      = "PUBLIC"
  table_name       = "EVENTS"
  dataset_label    = "Customer Events"
  dataset_kind     = "Event"
  valid_from_field = "EVENT_TIMESTAMP"

  depends_on = [
    snowflake_grant_privileges_to_share.schema,
    snowflake_grant_privileges_to_share.events,
  ]
}
