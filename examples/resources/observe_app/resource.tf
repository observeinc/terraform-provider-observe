resource "observe_folder" "example" {
  name = "OpenWeather"
}

resource "observe_datastream" "example" {
  name = "OpenWeather"
}

resource "observe_app" "example" {
  folder    = observe_folder.example.oid

  module_id = "observeinc/openweather/observe"
  version   = "0.2.1"

  variables = {
    datastream = observe_datastream.example.id
    api_key    = "..." # https://openweathermap.org/appid
  }
}
