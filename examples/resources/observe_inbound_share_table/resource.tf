# Basic table tracking
data "observe_inbound_share" "customer_data" {
  share_name       = "CUSTOMER_SHARE_PROD"
  provider_account = "ACME_CORP.US-EAST-1"
}

resource "observe_inbound_share_table" "customer_events" {
  share_id      = data.observe_inbound_share.customer_data.oid
  table_name    = "CUSTOMER_EVENTS"
  schema_name   = "PUBLIC"
  dataset_label = "Customer Events"
  dataset_kind  = "Table"
  description   = "Customer event data from Snowflake share"
}

# Event dataset with timestamp
resource "observe_inbound_share_table" "events" {
  share_id         = data.observe_inbound_share.customer_data.oid
  table_name       = "EVENTS"
  schema_name      = "PUBLIC"
  dataset_label    = "Events"
  dataset_kind     = "Event"
  valid_from_field = "event_timestamp"
  description      = "Event stream with timestamps"
}

# Interval dataset
resource "observe_inbound_share_table" "sessions" {
  share_id         = data.observe_inbound_share.customer_data.oid
  table_name       = "USER_SESSIONS"
  schema_name      = "PUBLIC"
  dataset_label    = "User Sessions"
  dataset_kind     = "Interval"
  valid_from_field = "session_start"
  valid_to_field   = "session_end"
}

# With field mappings
resource "observe_inbound_share_table" "metrics" {
  share_id      = data.observe_inbound_share.customer_data.oid
  table_name    = "METRICS"
  schema_name   = "PUBLIC"
  dataset_label = "Metrics"
  dataset_kind  = "Event"
  
  valid_from_field = "timestamp_ms"
  
  field_mapping {
    field      = "timestamp_ms"
    type       = "timestamp"
    conversion = "MillisecondsToTimestamp"
  }
  
  field_mapping {
    field      = "duration_ms"
    type       = "duration"
    conversion = "MillisecondsToDuration"
  }
}
