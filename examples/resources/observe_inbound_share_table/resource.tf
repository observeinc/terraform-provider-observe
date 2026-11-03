resource "observe_inbound_share" "customer_data" {
  share_name       = "CUSTOMER_SHARE_PROD"
  provider_account = "ACME_CORP.PROD"
}

# Reference table, without time fields
resource "observe_inbound_share_table" "customers" {
  share_id      = observe_inbound_share.customer_data.oid
  schema_name   = "PUBLIC"
  table_name    = "CUSTOMERS"
  dataset_label = "Customers"
  dataset_kind  = "Table"
  description   = "Customer records from the Snowflake share"
}

# Event dataset. Destroying the resource deletes the dataset, so guard it.
resource "observe_inbound_share_table" "events" {
  share_id         = observe_inbound_share.customer_data.oid
  schema_name      = "PUBLIC"
  table_name       = "EVENTS"
  dataset_label    = "Events"
  dataset_kind     = "Event"
  valid_from_field = "EVENT_TIMESTAMP"

  lifecycle {
    prevent_destroy = true
  }
}

# Interval dataset
resource "observe_inbound_share_table" "sessions" {
  share_id         = observe_inbound_share.customer_data.oid
  schema_name      = "PUBLIC"
  table_name       = "USER_SESSIONS"
  dataset_label    = "User Sessions"
  dataset_kind     = "Interval"
  valid_from_field = "SESSION_START"
  valid_to_field   = "SESSION_END"
}

# Event dataset whose time column holds epoch milliseconds
resource "observe_inbound_share_table" "metrics" {
  share_id         = observe_inbound_share.customer_data.oid
  schema_name      = "PUBLIC"
  table_name       = "METRICS"
  dataset_label    = "Metrics"
  dataset_kind     = "Event"
  valid_from_field = "TIMESTAMP_MS"

  field_mapping {
    field      = "TIMESTAMP_MS"
    type       = "timestamp"
    conversion = "MillisecondsToTimestamp"
  }
}
