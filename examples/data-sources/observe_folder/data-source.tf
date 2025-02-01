data "observe_workspace" "default" {
  name = "Default"
}

data "observe_folder" "lookup_by_name" {
  workspace = data.observe_workspace.default.oid
  name      = "name_of_a_folder"
}

data "observe_folder" "lookup_by_id" {
  id = "41000100"
}
