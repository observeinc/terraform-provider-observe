# Look up share by name and provider account (both required for uniqueness)
data "observe_inbound_share" "customer_data" {
  share_name       = "CUSTOMER_SHARE_PROD"
  provider_account = "ACME_CORP.US-EAST-1"
}

# Look up share by ID
data "observe_inbound_share" "by_id" {
  id = "41012345"
}

# Use in table tracking
resource "observe_inbound_share_table" "events" {
  share_id      = data.observe_inbound_share.customer_data.oid
  table_name    = "CUSTOMER_EVENTS"
  schema_name   = "PUBLIC"
  dataset_label = "Customer Events"
  dataset_kind  = "Event"
}

