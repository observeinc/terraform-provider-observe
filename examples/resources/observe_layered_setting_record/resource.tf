data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
}

# layered setting for a datastream retention for a specific datastream
resource "observe_layered_setting_record" "datastream_retention_example" {
    workspace     = data.observe_workspace.default.oid
    name          = "Layered Setting for retention 30 days example datastream"
    setting       = "DataRetention.periodDays"
    value_int64 = 30
    target        = data.observe_datastream.example.oid
}
