data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example"
}

resource "observe_poller" "weather" {
  workspace = data.observe_workspace.default.oid
  interval  = "5m"
  name      = "OpenWeather"

  datastream = observe_datastream.example.oid

  http {
    template {
      url = "https://api.openweathermap.org/data/2.5/weather"
      params = {
        units = "metric"
        appid = "my-api-key"
      }
    }

    request {
      params = {
        q = "San Francisco"
      }
    }

    request {
      params = {
        q = "New York"
      }
    }
  }
}
