data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Dashboard"
}

resource "observe_default_dashboard_for_tag" "example" {
  tag       = "k8s.namespace.name"
  dashboard = observe_dashboard.example.oid
}
