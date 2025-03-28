resource "observe_dashboard" "example" {
  name      = "example"
  stages    = jsonencode(
    [
      {
        id       = "stage-nkeju1il"
        input    = [
          {
            datasetId   = "41000014"
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
}
