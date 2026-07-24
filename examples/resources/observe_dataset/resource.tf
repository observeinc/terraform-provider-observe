data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "observation" {
  workspace = data.observe_workspace.default.oid
  name      = "Observation"
}

resource "observe_dataset" "http_observations" {
  workspace = data.observe_workspace.default.oid
  name      = "HTTP observations"

  inputs = {
    "observation" = data.observe_dataset.observation.oid
  }

  stage {
    pipeline = <<-EOT
      filter OBSERVATION_KIND = "http"
    EOT
  }

  # Optional: Object tags for organizing and categorizing datasets
  object_tags = {
    environment = "production"
    team        = "backend,frontend"
    category    = "observability"
  }
}
