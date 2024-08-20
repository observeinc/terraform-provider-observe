data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastet" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Dataset"
}
