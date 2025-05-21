---
subcategory: ""
page_title: "How to lock down terraform-managed content"
description: |-
  How to lock down terraform-managed content
---

## How to lock down terraform-managed content

### Prerequisites

- Ensure there’s a separate user specifically for terraform that’s not tied to any real user.
  - This also ensures that terraform doesn't stop working when an employee leaves.
  - For easy identification of this user in the UI, name it "Terraform ..."
- Ensure [audit logs](https://docs.observeinc.com/en/latest/content/reference/rbac/auditTrail.html) are enabled and Usage app version ≥0.24.0 is installed.

### Restricting edit access through RBAC

Let's lock down the following dashboard defined in terraform:

```terraform
resource "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
  stages    = jsonencode([])
}
```

Use [`observe_resource_grants`](https://registry.terraform.io/providers/observeinc/observe/latest/docs/resources/resource_grants) to lock down permissions like so:

```terraform
data "observe_user" "terraform" {
  email = "<terraform_user_email_address>"
}

data "observe_rbac_group" "everyone" {
  name = "Everyone"
}

resource "observe_resource_grants" "lock_down_dashboard_example" {
  oid = observe_dashboard.example.oid
  grant {
    subject = data.observe_user.terraform.oid
    role    = "dashboard_editor"
  }
  grant {
    subject = data.observe_rbac_group.everyone.oid
    role    = "dashboard_viewer"
  }
}
```

`observe_resource_grants` is authoritative and controls the entire set of grants for the resource. So the above will effectively delete all existing grants for the example dashboard and share it *only* with the specified subjects. This ensures that no one except the terraform user has permission to edit the dashboard. We also retain view access for the group Everyone, which can be replaced with the desired set of groups who should have view access.

To lock down many resources, use the `for_each` meta-argument, for example:

```terraform
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
```

~> **NOTE:** Users with administrator privileges will always be able to edit all resources.

Not enough?
- Want to ensure admins don't accidentally edit terraform-managed content?
- Want to allow edits of terraform-managed content in the UI, but get notified of it to ensure those changes are persisted in terraform?

Then read on!

### Alerting on edits occurring outside of terraform

Add the following to monitor non-terraform updates to resources that were last updated by terraform.

```terraform
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
```

We use `updated_by` here to ensure this works for resources that were created through the UI and imported into terraform. Can also use `created_by` if that’s not a concern.