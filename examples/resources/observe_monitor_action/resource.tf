data "observe_workspace" "default" {
  name = "Default"
}

# Monitor action that triggers a webhoop to example.com with the http header test set to
# hello
resource "observe_monitor_action" "webhook_action" {
  workspace = data.observe_workspace.default.oid
  name      = "%s"
  icon_url  = "test"

  webhook {
    url_template = "https://example.com"
    body_template = "{}"
    headers = {
      "test" = "hello"
    }
  }
}

# email monitor action
resource "observe_monitor_action" "email_action" {
  workspace = data.observe_workspace.default.oid
  name      = "%s"
  icon_url  = "test"

  email {
    target_addresses = [ "test@observeinc.com" ]
  subject_template = "Hello"
  body_template    = "Nope"
  }
}
