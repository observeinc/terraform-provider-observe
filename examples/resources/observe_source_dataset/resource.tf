# Native (Observe-managed) storage. storage and schema_type default to Native and Any.
resource "observe_source_dataset" "logs" {
  label       = "myapp/Logs"
  description = "Raw application logs"

  object_tags = {
    team = "platform"
  }
}

# Observe-managed Iceberg table in a storage integration.
resource "observe_source_dataset" "iceberg_logs" {
  label               = "myapp/Iceberg Logs"
  schema_type         = "OtelLogs"
  storage             = "Iceberg"
  storage_integration = "o:::storageintegration:41000123"
}

# External write: Observe provisions the Iceberg table, and you write rows
# through Observe's Iceberg catalog.
resource "observe_source_dataset" "external" {
  label               = "myapp/External Events"
  storage             = "Iceberg"
  storage_integration = "o:::storageintegration:41000123"
  external_write      = true
}
