resource "observe_user" "example" {
  email             = "jane.doe@example.com"
  label             = "Jane Doe"
  identity_provider = "SAML2"
  # Must match the NameID your IdP sends at login.
  identity = "jane.doe@example.com"
}

data "observe_rbac_group" "engineering" {
  name = "engineering"
}

resource "observe_rbac_group_member" "example" {
  group = data.observe_rbac_group.engineering.oid
  member {
    user = observe_user.example.oid
  }
}
