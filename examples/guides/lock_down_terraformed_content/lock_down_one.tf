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
