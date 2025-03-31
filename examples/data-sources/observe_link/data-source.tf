data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "a" {
  workspace = data.observe_workspace.default.oid
  name      = "Dataset A"
}

data "observe_dataset" "b" {
  workspace = data.observe_workspace.default.oid
  name      = "Dataset B"
}

# check that link from dataset a to dataset b on key "key" exists
data "observe_link" "check" {
  source = data.observe_dataset.a.oid
  target = data.observe_dataset.b.oid
  fields = ["key"]
}
