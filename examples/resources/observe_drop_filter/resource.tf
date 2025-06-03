data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Datastream"
}

### example 1 ###

resource "observe_drop_filter" "example" {
  workspace      = data.observe_workspace.default.oid
  name           = "test-filter"
  pipeline       = "filter FIELDS.x ~ y"
  source_dataset = data.observe_datastream.example.dataset
  drop_rate      = 0.99
  enabled        = true
}

### example 2 ###

data "observe_dataset" "kubernetes" {
  workspace = data.observe_workspace.default.oid
  name      = "Kubernetes Explorer/OpenTelemetry Logs"
}

resource "observe_drop_filter" "example2" {
  workspace      = data.observe_workspace.default.oid
  name           = "test-filter-2"
  pipeline       = <<PIPE
    filter resource_attributes."k8s.container.name" = "image-provider"
  PIPE
  source_dataset = data.observe_dataset.kubernetes.oid
  drop_rate      = 0.99
  # enabled is an optional field and defaults to true
}
