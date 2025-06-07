data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_ingest_token" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Ingest Token"
}
