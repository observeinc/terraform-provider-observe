data "observe_workspace" "default" {
  name = "Default"
}

data "observe_user" "example" {
  email = "example@domain.com"
}

data "observe_rbac_group" "example" {
  name = "engineering"
}

resource "observe_rbac_statement" "user_example" {
  description = "Allow user access to workspace contents"
  subject {
    user = data.observe_user.example.oid
  }
  object {
    workspace = data.observe_workspace.default.id
  }
  role = "Viewer"
}

resource "observe_rbac_statement" "group_example" {
  description = "Allow group access to workspace contents"
  subject {
    group = data.observe_rbac_group.example.oid
  }
  object {
    workspace = data.observe_workspace.default.id
  }
  role = "Viewer"
}
