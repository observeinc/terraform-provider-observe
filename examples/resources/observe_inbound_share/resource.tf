resource "observe_inbound_share" "customer_data" {
  share_name       = "CUSTOMER_SHARE_PROD"
  provider_account = "ACME_CORP.PROD"
  label            = "Customer data"

  # The share comment must hold the registration token before registering
  depends_on = [snowflake_share.customer_data]
}

resource "observe_inbound_share_table" "events" {
  share_id         = observe_inbound_share.customer_data.oid
  table_name       = "EVENTS"
  schema_name      = "PUBLIC"
  dataset_label    = "Customer Events"
  dataset_kind     = "Event"
  valid_from_field = "EVENT_TIMESTAMP"
}
