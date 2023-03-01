data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Dataset"
}

resource "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Dashboard"
}

resource "observe_default_dashboard" "example" {
  dataset   = observe_dataset.example.oid
  dashboard = observe_dashboard.example.oid
}
