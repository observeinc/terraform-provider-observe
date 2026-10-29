resource "observe_composite_monitor" "example" {
  name        = "API errors without a deploy"
  description = "Errors are elevated and no deploy is in progress."
  condition   = "errors && !deploying"

  constituents {
    alias   = "errors"
    monitor = observe_monitor_v2.api_errors.oid
    # let the composite be the only thing that notifies for this monitor
    notify_independently = false
  }

  constituents {
    alias   = "deploying"
    monitor = observe_monitor_v2.deploy_in_progress.oid
  }

  actions {
    oid    = observe_monitor_v2_action.oncall.oid
    levels = ["critical", "error"]
  }

  actions {
    action {
      type = "webhook"
      webhook {
        url    = "https://example.com/hook"
        method = "post"
        body   = jsonencode({ text = "API errors outside a deploy" })
      }
    }
    send_end_notifications  = true
    send_reminders_interval = "30m"
  }

  custom_variables = {
    title = "API errors outside a deploy"
  }
}
