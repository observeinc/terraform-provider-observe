resource "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
  stages    = jsonencode([])
}
