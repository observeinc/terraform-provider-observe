data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Datastream"
}

# Simplest possible log-derived metric: only required fields
resource "observe_log_derived_metric_dataset" "simple_count" {
  workspace = data.observe_workspace.default.oid
  name      = "Request Count"

  metric_name = "request_count"

  input = observe_datastream.example.dataset

  aggregation {
    function = "count"
  }
}

# Log-derived metric counting errors per service
resource "observe_log_derived_metric_dataset" "error_count" {
  workspace   = data.observe_workspace.default.oid
  name        = "Error Count Metric"
  description = "Counts error log lines per service"

  metric_name = "error_count"
  metric_type = "gauge"
  unit        = "1"
  interval    = "1m"

  input = observe_datastream.example.dataset
  query = "filter severity = \"ERROR\""

  aggregation {
    function = "count"
  }

  metric_tag {
    name   = "service"
    column = "service_name"
  }

  metric_tag {
    name   = "environment"
    column = "env"
  }
}

# Log-derived metric with sum aggregation and a multiline query
resource "observe_log_derived_metric_dataset" "total_bytes" {
  workspace   = data.observe_workspace.default.oid
  name        = "Total Bytes Transferred"
  description = "Sum of bytes transferred per service"

  metric_name = "bytes_transferred"
  metric_type = "cumulative_counter"
  unit        = "bytes"
  interval    = "5m"

  input = observe_datastream.example.dataset
  query = <<-EOT
    filter status_code >= 200 and status_code < 300
    filter content_type = "application/json"
  EOT

  aggregation {
    function = "sum"
    field_path {
      column = "bytes_sent"
    }
  }

  metric_tag {
    name   = "service"
    column = "service_name"
  }
}

# Log-derived metric with average aggregation and multiple tags
resource "observe_log_derived_metric_dataset" "avg_response_time" {
  workspace = data.observe_workspace.default.oid
  name      = "Average Response Time"

  metric_name = "response_time_avg"
  metric_type = "gauge"
  unit        = "milliseconds"
  interval    = "1m"

  input = observe_datastream.example.dataset
  query = "filter endpoint != \"/health\""

  aggregation {
    function = "avg"
    field_path {
      column = "response_time_ms"
    }
  }

  metric_tag {
    name   = "endpoint"
    column = "endpoint"
  }

  metric_tag {
    name   = "method"
    column = "http_method"
  }
}

# Log-derived metric with count_distinct aggregation
resource "observe_log_derived_metric_dataset" "unique_users" {
  workspace = data.observe_workspace.default.oid
  name      = "Unique Active Users"

  metric_name = "active_users"
  metric_type = "gauge"
  unit        = "users"
  interval    = "10m"

  input = observe_datastream.example.dataset
  query = "filter action = \"login\""

  aggregation {
    function = "count_distinct"
    field_path {
      column = "user_id"
    }
  }

  metric_tag {
    name   = "region"
    column = "region"
  }
}
