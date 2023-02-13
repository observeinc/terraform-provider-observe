data "observe_folder" "example" {
  name = "OpenWeather"
}

data "observe_app" "example" {
  folder = data.observe_folder.example.oid
  name   = "OpenWeather"
}