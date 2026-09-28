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

  # Optional: Object tags for organizing and categorizing dashboards
  object_tags = {
    team       = "platform"
    visibility = "public,internal"
  }
}

# A schema_version = 2 dashboard is described entirely by a single JSON
# `definition` document: a layout of titled sections holding placed cards. It is
# created and updated through the REST API, and `stages`, `layout`, and
# `parameters` must not be set. Updates send only the fields that changed
# (merge-patch semantics).
#
# Alpha: managing a dashboard through `definition` (schema_version >= 2) is an alpha
# feature that must be enabled for your account and may change in
# backwards-incompatible ways. Not recommended for production dashboards.
resource "observe_dashboard" "example_rest" {
  name           = "example-rest"
  schema_version = 2
  definition = jsonencode({
    layout = {
      sections = [
        {
          title = "Overview"
          cards = [
            {
              type     = "query"
              geometry = { x = 0, y = 0, w = 12, h = 6 }
              query = {
                content = {
                  pipeline = [
                    "filter label(^Trace) ~ 'foo'",
                    "filter event_name = \"event 1\"",
                  ]
                  inputs = [
                    {
                      name = "OpenTelemetry/Span Event"
                      source = {
                        type    = "dataset"
                        dataset = { id = data.observe_dataset.span_event.id }
                      }
                    },
                  ]
                }
              }
            },
          ]
        },
      ]
    }
  })

  # Optional: Object tags for organizing and categorizing dashboards
  object_tags = {
    team       = "platform"
    visibility = "public,internal"
  }
}
