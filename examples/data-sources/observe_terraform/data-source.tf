data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "kubernetes_container" {
  workspace = data.observe_workspace.default.oid
  name      = "kubernetes/Container"
}

data "observe_terraform" "Example" {
  target = data.observe_dataset.kubernetes_container.oid
}
