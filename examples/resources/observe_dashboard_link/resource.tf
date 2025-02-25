
data "observe_workspace" "default" {
  name = "Default"
}

data "observe_folder" "example_folder" {
  workspace = data.observe_workspace.default.oid
  name      = "name_of_a_folder"
}

data "observe_dashboard" "a" {
  id = "41000100"
}

data "observe_dashboard" "b" {
  id = "41000101"
}

# Can either link by specifying the folder
resource "observe_dashboard_link" "example_with_folder" {
  folder         = data.observe_folder.example_folder.oid
  name           = "Dashboard Link (Folder)"
  description    = "Very linked, much dashboard"
  from_dashboard = data.observe_dashboard.a.oid
  to_dashboard   = data.observe_dashboard.b.oid
  from_card      = "some card"
  link_label     = "go hither"
}

# ...or by specifying the workspace
resource "observe_dashboard_link" "example_with_workspace" {
  workspace      = data.observe_workspace.default.oid
  name           = "Dashboard Link (Workspace)"
  description    = "Very linked, much dashboard"
  from_dashboard = data.observe_dashboard.b.oid
  to_dashboard   = data.observe_dashboard.a.oid
  link_label     = "go yon"
}
