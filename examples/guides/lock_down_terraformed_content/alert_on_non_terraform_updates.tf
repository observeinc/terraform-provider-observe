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

data "observe_dataset" "dataset" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Dataset"
}

data "observe_dataset" "datastream" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Datastream"
}

data "observe_dataset" "monitor" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Monitor"
}

data "observe_dataset" "worksheet" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Worksheet"
}

resource "observe_monitor_v2" "update-to-terraformed-content" {
  workspace = data.observe_workspace.default.oid
  name      = "Non-terraform update to terraformed content"
  rule_kind = "promote"

  inputs = {
    "usage/Audit Events"       = data.observe_dataset.audit_events.oid
    "usage/Observe Dashboard"  = data.observe_dataset.dashboard.oid
    "usage/Observe Dataset"    = data.observe_dataset.dataset.oid
    "usage/Observe Datastream" = data.observe_dataset.datastream.oid
    "usage/Observe Monitor"    = data.observe_dataset.monitor.oid
    "usage/Observe Worksheet"  = data.observe_dataset.worksheet.oid
  }

  stage {
    input    = "usage/Audit Events"
    pipeline = <<-EOT
            filter operation = "update"
            
            lookup ^Dataset, dataset_last_updated_by:@"usage/Observe Dataset".updated_by
            lookup ^Monitor, monitor_last_updated_by:@"usage/Observe Monitor".updated_by
            lookup ^Datastream, datastream_last_updated_by:@"usage/Observe Datastream".updated_by
            lookup ^Dashboard, dashboard_last_updated_by:@"usage/Observe Dashboard".updated_by
            lookup ^Worksheet, worksheet_last_updated_by:@"usage/Observe Worksheet".updated_by
            
            make_col last_updated_by:coalesce(
                dataset_last_updated_by,
                monitor_last_updated_by,
                datastream_last_updated_by,
                dashboard_last_updated_by,
                worksheet_last_updated_by
              )
            
            filter last_updated_by = ${data.observe_user.terraform.id} and user_id != ${data.observe_user.terraform.id}
        EOT
  }

  rules {
    level = "warning"
    promote {}
  }
}
