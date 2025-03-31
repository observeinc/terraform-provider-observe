data "observe_workspace" "default" {
  name = "Default"
}

# lookup by id
data "observe_monitor_action" "id_lookup" {
  id = 41000100
}

# lookup by name
data "observe_monitor_action" "name_lookup" {
  workspace = data.observe_workspace.default.oid
  name      = "name of the monitor action"
}
