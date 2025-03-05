data "observe_rbac_group" "engineering" {
  name = "engineering"
}

data "observe_rbac_group" "readonly" {
  name = "readonly"
}

// Allow group "engineering" to edit and group "readonly" to view newly created resources by default.
resource "observe_workspace_default_grants" "example" {
  group {
    oid        = data.observe_rbac_group.engineering.oid
    permission = "edit"
  }

  group {
    oid        = data.observe_rbac_group.readonly.oid
    permission = "view"
  }
}

// Only the creating user (and admins) can edit newly created resources by default.
resource "observe_workspace_default_grants" "empty" {}

// Allow group "engineering" to edit newly created dashboards and worksheets by default, but only
// view datastreams. Allow group "readonly" to still view all newly created resources by default.
resource "observe_workspace_default_grants" "limited" {
  group {
    oid          = data.observe_rbac_group.engineering.oid
    permission   = "edit"
    object_types = ["dashboard", "worksheet"]
  }

  group {
    oid          = data.observe_rbac_group.engineering.oid
    permission   = "view"
    object_types = ["datastream"]
  }

  group {
    oid        = data.observe_rbac_group.readonly.oid
    permission = "view"
  }
}
