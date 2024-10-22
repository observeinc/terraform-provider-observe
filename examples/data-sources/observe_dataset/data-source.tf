data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Dataset"
}
