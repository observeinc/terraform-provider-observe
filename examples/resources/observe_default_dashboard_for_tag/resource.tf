resource "observe_dashboard" "example" {
  name = "Example Dashboard"
}

resource "observe_default_dashboard_for_tag" "example" {
  tag       = "k8s.namespace.name"
  dashboard = observe_dashboard.example.oid
}
