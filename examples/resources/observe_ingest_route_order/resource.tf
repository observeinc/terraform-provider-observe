resource "observe_ingest_route" "logs" {
  type           = "otellogs"
  pipeline       = "filter FIELDS.service == \"api\""
  destination_id = "41007777"
}

resource "observe_ingest_route_order" "logs" {
  type = "otellogs"

  route_ids = [
    observe_ingest_route.logs.route_id,
    "41030002",
  ]
}