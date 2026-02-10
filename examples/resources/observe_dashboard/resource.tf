data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "span_event" {
  workspace = data.observe_workspace.default.oid
  name      = "OpenTelemetry/Span Event"
}

resource "observe_dashboard" "example" {
  name = "example"
  stages = jsonencode(
    [
      {
        id = "stage-nkeju1il"
        input = [
          {
            datasetId   = data.observe_dataset.span_event.id
            datasetPath = null
            inputName   = "OpenTelemetry/Span Event"
            inputRole   = "Data"
            stageId     = null
          },
        ]
        params   = null
        pipeline = <<-EOT
          filter label(^Trace) ~ 'foo'
          filter event_name = "event 1"
        EOT
      },
    ]
  )
  workspace = data.observe_workspace.default.oid

  # Optional: Entity tags for organizing and categorizing dashboards
  entity_tags = {
    team       = "platform"
    visibility = "public,internal"
  }
}
