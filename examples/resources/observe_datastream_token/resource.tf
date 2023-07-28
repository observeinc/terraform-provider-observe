data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Datastream"
}

resource "observe_datastream_token" "example" {
  datastream = data.observe_datastream.example.oid
  name       = "My Token"
}
