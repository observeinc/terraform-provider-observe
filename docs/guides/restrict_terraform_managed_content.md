---
subcategory: ""
page_title: "Restrict edit access for terraform managed content"
description: |-
  Restrict edit access for terraform managed content
---

## Restrict edit access for terraform managed content

This page walks through how to set up RBAC rules to ensure no one can directly edit content that's managed by terraform.
We focus on dashboards here, but the same principles apply to all other resources that can be targeted through RBAC.

~> **NOTE:** Users with administrator privileges will always be able to edit **all** resources. If this is a concern, see [Alerting on undesired edits to terraform managed content](#alerting-on-undesired-edits-to-terraform-managed-content). These alerts can also be used in place of restricting edit access if you want to allow using the UI for edits, while still being notified of such changes to ensure they're persisted in terraform and not overwritten.

### Prerequisites
Before proceeding, ensure there’s a separate user specifically for terraform that’s not tied to any real user.
- This gives us a way to identify which operations are performed by terraform, and is a good practice in general to ensure terraform doesn't stop working when a user is disabled in Observe.

### Restricting access through RBAC

Let's lock down the following dashboard defined in terraform:

```terraform
resource "observe_dashboard" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
  stages    = jsonencode([])
}
```

Use [`observe_resource_grants`](https://registry.terraform.io/providers/observeinc/observe/latest/docs/resources/resource_grants) to lock down permissions:

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

`observe_resource_grants` is authoritative and controls the entire set of grants for the resource. So the above will effectively delete all existing grants for the example dashboard and share it with *only* the specified subjects. This ensures that no one except the terraform user has permission to edit the dashboard. We also retain view access for the group Everyone, which can be replaced with the desired set of groups who should have view access.

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

### Alerting on undesired edits to terraform managed content
Since admin users are not subject to RBAC restrictions, alerting can be a useful fallback to ensure an admin's edits to terraform managed content are not lost when terraform next runs.

⚠️ Before proceeding, ensure [audit logs](https://docs.observeinc.com/en/latest/content/reference/rbac/auditTrail.html) are enabled and Usage app version ≥0.24.0 is installed.

Add the following to monitor non-terraform updates to resources that were last updated through terraform:

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
```

We use `updated_by` here to ensure this works for resources that were created through the UI and imported into terraform. Can also use `created_by` if that’s not a concern.

Can monitor additional resources by updating the OPAL to include them, as well as adding the corresponding inputs.

```
filter operation = "update" and in(object_kind, "dashboard", "dataset")
lookup ^Dashboard, dashboard_last_updated_by:@"usage/Observe Dashboard".updated_by
lookup ^Dataset, dataset_last_updated_by:@"usage/Observe Dataset".updated_by
make_col last_updated_by:coalesce(dashboard_last_updated_by, dataset_last_updated_by)
filter last_updated_by = ${data.observe_user.terraform.id} and user_id != ${data.observe_user.terraform.id}
```
