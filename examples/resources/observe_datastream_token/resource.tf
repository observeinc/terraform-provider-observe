data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Datastream"
}
