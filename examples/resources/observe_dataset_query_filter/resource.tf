data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Dataset"
}

resource "observe_dataset_query_filter" "example" {
  dataset     = data.observe_dataset.example.oid
  label       = "PII Filter"
  description = "Filter to exclude rows containing personally identifiable information"
  filter      = "body ~ <phoneNumber> and body ~ <emailAddress>"
  disabled    = false
}
