data "observe_user" "example" {
  email = "example@domain.com"
}

resource "observe_rbac_group" "example" {
  name = "engineering"
}

resource "observe_rbac_group_member" "example" {
  group = observe_rbac_group.example.oid
  member {
    user = data.observe_user.example.oid
  }
}

data "observe_rbac_group_member" "example" {
  id = observe_rbac_group_member.example.id
}
