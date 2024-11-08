data "observe_folder" "example" {
  workspace = data.observe_workspace.default
  name      = "OpenWeather"
}

data "observe_app" "example" {
  folder = data.observe_folder.example.oid
  name   = "OpenWeather"
}
