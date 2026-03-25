data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "Example Datastream"
}

resource "observe_log_derived_metric_dataset" "error_count" {
  workspace   = data.observe_workspace.default.oid
  name        = "Error Count Metric"
  description = "Counts error log lines per service"

  metric_name = "error_count"
  metric_type = "gauge"
  unit        = "1"
  interval    = "1m"

  shaping_query {
    inputs = {
      "logs" = observe_datastream.example.dataset
    }
    pipeline = "filter severity = \"ERROR\""
  }

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

# Log-derived metric with sum aggregation on a field
resource "observe_log_derived_metric_dataset" "total_bytes" {
  workspace   = data.observe_workspace.default.oid
  name        = "Total Bytes Transferred"
  description = "Sum of bytes transferred per service"

  metric_name = "bytes_transferred"
  metric_type = "cumulative_counter"
  unit        = "bytes"
  interval    = "5m"

  shaping_query {
    inputs = {
      "access_logs" = observe_datastream.example.dataset
    }
    pipeline = "filter status_code >= 200 and status_code < 300"
  }

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

  shaping_query {
    inputs = {
      "api_logs" = observe_datastream.example.dataset
    }
    pipeline = "filter endpoint != \"/health\""
  }

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

  shaping_query {
    inputs = {
      "activity" = observe_datastream.example.dataset
    }
    pipeline = "filter action = \"login\""
  }

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
