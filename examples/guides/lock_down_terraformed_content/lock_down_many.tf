resource "observe_dashboard" "a" {
  workspace = data.observe_workspace.default.oid
  name      = "a"
  stages    = jsonencode([])
}

resource "observe_dashboard" "b" {
  workspace = data.observe_workspace.default.oid
  name      = "b"
  stages    = jsonencode([])
}

resource "observe_dashboard" "c" {
  workspace = data.observe_workspace.default.oid
  name      = "c"
  stages    = jsonencode([])
}

locals {
  dashboards = [
    observe_dashboard.a.oid,
    observe_dashboard.b.oid,
    observe_dashboard.c.oid,
  ]
}

resource "observe_resource_grants" "lock-down-dashboards" {
  for_each = toset(local.dashboards)

  oid = each.value
  grant {
    subject = data.observe_user.terraform.oid
    role    = "dashboard_editor"
  }
  grant {
    subject = data.observe_rbac_group.everyone.oid
    role    = "dashboard_viewer"
  }
}
