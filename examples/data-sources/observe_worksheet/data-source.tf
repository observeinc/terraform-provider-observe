data "observe_workspace" "default" {
  name = "Default"
}

data "observe_worksheet" "lookup" {
  workspace = data.observe_workspace.default.oid
  id        = "41000100"
}
