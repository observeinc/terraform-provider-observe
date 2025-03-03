data "observe_rbac_group" "engineering" {
  name = "engineering"
}

data "observe_rbac_group" "readonly" {
  name = "readonly"
}

// Allow group "engineering" to edit and group "readonly" to view newly created resources by default.
// Only one of this resource can exist in a given tenant.
resource "observe_default_sharing_groups" "example" {
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
resource "observe_default_sharing_groups" "empty" {}
