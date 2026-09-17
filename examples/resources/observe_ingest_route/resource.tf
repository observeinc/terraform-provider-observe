resource "observe_ingest_route" "logs" {
  type           = "otellogs"
  pipeline       = "filter FIELDS.service == \"api\""
  destination_id = "41007777"
  enabled        = true
}

resource "observe_ingest_route" "logs_copy" {
  type                     = "otellogs"
  pipeline                 = "filter FIELDS.service == \"billing\""
  destination_id           = "41007777"
  secondary_destination_id = "41008888"
}