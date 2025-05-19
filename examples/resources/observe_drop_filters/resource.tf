data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Datastream"
}


resource "observe_drop_filter" "example" {
  workspace      = data.observe_workspace.default.oid
  name           = "test-filter"
  pipeline       = "filter FIELDS.x ~ y"
  source_dataset = data.observe_datastream.example.dataset
  drop_rate      = 0.99
  enabled        = true
}
