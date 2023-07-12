data "observe_user" "example" {
  email = "example@domain.com"
}

data "observe_rbac_group" "reader" {
  name = "reader"
}

data "observe_rbac_group" "example" {
  name = "engineering"
}

resource "observe_rbac_group_member" "user_example" {
  group       = data.observe_rbac_group.reader.oid
  description = "add example user to reader group"
  member {
    user = data.observe_user.example.oid
  }
}

resource "observe_rbac_group_member" "group_example" {
  group       = data.observe_rbac_group.reader.oid
  description = "add example group to reader group"
  member {
    group = data.observe_rbac_group.example.oid
  }
}
