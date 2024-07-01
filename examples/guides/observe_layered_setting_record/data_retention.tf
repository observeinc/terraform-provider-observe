# layered setting for a datastream retention for a specific stream
# targets a datastream OID
resource "observe_layered_setting_record" "datastream_retention_my_first" {
  workspace   = data.observe_workspace.default.oid
  name        = "Layered Setting For Retention 30 days my first datastream"
  setting     = "DataRetention.periodDays"
  value_int64 = 30
  target      = resource.observe_datastream.my_first_datastream.oid
}
