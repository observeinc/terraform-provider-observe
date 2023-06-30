data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_bookmark_group" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
}

# Dataset

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
}

resource "observe_bookmark" "dataset" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
  target    = data.observe_dataset.example.oid
}

# Dashboard

data "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
}

resource "observe_bookmark" "dashboard" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
  target    = data.observe_dashboard.example.oid
}