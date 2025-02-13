data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "a" {
  workspace = data.observe_workspace.default.oid
  name      = "Dataset A"
}

# query on dataset A
data "observe_query" "query_for_a" {
  start = timestamp()

  inputs = { "test" = data.observe_dataset.a.oid }

  stage {
    pipeline = <<-EOF
      # ... OPAL query
    EOF
  }
}
