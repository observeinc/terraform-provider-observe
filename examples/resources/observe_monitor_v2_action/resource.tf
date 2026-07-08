# shared webhook action
resource "observe_monitor_v2_action" "webhook_action" {
  name = "PagerDuty Alert"
  type = "webhook"

  webhook {
    url    = "https://events.pagerduty.com/v2/enqueue"
    method = "post"
    body   = "{\"routing_key\": \"YOUR_ROUTING_KEY\", \"event_action\": \"trigger\"}"
    headers {
      header = "Content-Type"
      value  = "application/json"
    }
  }
}

# shared email action
resource "observe_monitor_v2_action" "email_action" {
  name        = "Notify On-Call"
  type        = "email"
  description = "Sends an alert email to the on-call team."

  email {
    subject   = "Monitor Alert"
    body      = "A monitor has triggered. Check the Observe console for details."
    addresses = ["oncall@example.com"]
  }
}

# attach a shared action to a monitor
resource "observe_monitor_v2" "example" {
  # ... monitor configuration ...

  actions {
    oid    = observe_monitor_v2_action.webhook_action.oid
    levels = ["critical", "error"]
  }
}

# alternatively, define an inline action directly on a monitor
resource "observe_monitor_v2" "inline_example" {
  # ... monitor configuration ...

  actions {
    levels = ["warning"]
    action {
      type = "email"
      email {
        subject   = "Warning Alert"
        addresses = ["team@example.com"]
      }
    }
  }
}
