data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "a" {
  workspace = data.observe_workspace.default.oid
  name      = "Dataset A"
}

resource "observe_correlation_tag" "example" {
  name = "%[1]s-key.name"
  dataset = observe_dataset.a.oid
  # tag the dataset for correlation using its "key" column
  column = "key"
}
