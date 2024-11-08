data "observe_workspace" "default" {
  name = "Default"
}

data "observe_user" "example" {
  email = "example@domain.com"
}

data "observe_rbac_group" "example" {
  name = "engineering"
}

// "Everyone" is a special pre-defined group that always includes all users
data "observe_rbac_group" "everyone" {
  name = "Everyone"
}

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Engineering Logs"
}

// Allow user example to create worksheets
resource "observe_grant" "user_example" {
  subject = data.observe_user.example.oid
  role    = "worksheet_creator"
}

// Allow group engineering to edit dataset Engineering Logs
resource "observe_grant" "group_example" {
  subject = data.observe_rbac_group.example.oid
  role    = "dataset_editor"
  qualifier {
    oid = data.observe_dataset.example.oid
  }
}

// Allow everyone to view dataset Engineering Logs
resource "observe_grant" "everyone_example" {
  subject = data.observe_rbac_group.everyone.oid
  role    = "dataset_viewer"
  qualifier {
    oid = data.observe_dataset.example.oid
  }
}
