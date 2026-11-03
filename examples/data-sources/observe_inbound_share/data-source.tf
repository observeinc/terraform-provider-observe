# Look up a share by its Snowflake name and provider account
data "observe_inbound_share" "customer_data" {
  snowflake_config {
    share_name       = "CUSTOMER_SHARE_PROD"
    provider_account = "ACME_CORP.PROD"
  }
}

# Look up a share by ID
data "observe_inbound_share" "by_id" {
  id = "41012345"
}

# Track a table from a share registered outside this configuration
resource "observe_inbound_share_table" "events" {
  share_id         = data.observe_inbound_share.customer_data.oid
  schema_name      = "PUBLIC"
  table_name       = "CUSTOMER_EVENTS"
  dataset_label    = "Customer Events"
  dataset_kind     = "Event"
  valid_from_field = "EVENT_TIMESTAMP"
}
