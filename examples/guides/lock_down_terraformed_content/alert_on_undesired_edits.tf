data "observe_workspace" "default" {
  name = "Default"
}

data "observe_user" "terraform" {
  email = "<terraform_user_email_address>"
}

data "observe_dataset" "audit_events" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Audit Events"
}

data "observe_dataset" "dashboard" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Dashboard"
}

resource "observe_monitor_v2" "update-to-terraformed-content" {
  workspace = data.observe_workspace.default.oid
  name      = "Non-terraform update to terraformed content"
  rule_kind = "promote"

  inputs = {
    "usage/Audit Events"      = data.observe_dataset.audit_events.oid
    "usage/Observe Dashboard" = data.observe_dataset.dashboard.oid
  }

  stage {
    input    = "usage/Audit Events"
    pipeline = <<-EOT
            filter operation = "update" and object_kind = "dashboard"
            lookup ^Dashboard, last_updated_by:@"usage/Observe Dashboard".updated_by
            filter last_updated_by = ${data.observe_user.terraform.id} and user_id != ${data.observe_user.terraform.id}
        EOT
  }

  rules {
    level = "warning"
    promote {}
  }
}
