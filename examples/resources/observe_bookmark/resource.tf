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
  group  = observe_bookmark_group.example.oid
  name   = "Example"
  target = data.observe_dataset.example.oid

  # Optional: Object tags for organizing and categorizing bookmarks
  object_tags = {
    category = "monitoring"
    priority = "high,critical"
  }
}

# Dashboard

data "observe_dashboard" "example" {
  id = "<Dashboard_ID>"
}

resource "observe_bookmark" "dashboard" {
  group  = observe_bookmark_group.example.oid
  name   = "Example"
  target = data.observe_dashboard.example.oid
}