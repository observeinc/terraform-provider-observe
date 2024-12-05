data "observe_workspace" "default" {
  name = "Default"
}

data "observe_rbac_group" "example" {
  name = "engineering"
}

data "observe_rbac_group" "everyone" {
  name = "Everyone"
}

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Engineering Logs"
}

data "observe_dataset" "example2" {
  workspace = data.observe_workspace.default.oid
  name      = "Secrets"
}

// Allow group engineering to edit and Everyone to view dataset Engineering Logs.
// Ensures there are no other grants targeting this dataset,
// so no one else (except admins) can view or edit it.
resource "observe_resource_grants" "example" {
  oid = data.observe_dataset.example.oid
  grant {
    subject = data.observe_rbac_group.example.oid
    role    = "dataset_editor"
  }
  grant {
    subject = data.observe_rbac_group.everyone.oid
    role    = "dataset_viewer"
  }
}


// Only allow admins to view or edit the dataset Secrets
resource "observe_resource_grants" "example2" {
  oid = data.observe_dataset.example2.oid
}
