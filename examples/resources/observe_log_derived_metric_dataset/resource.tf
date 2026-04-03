# Basic log derived metric dataset with count aggregation
resource "observe_log_derived_metric_dataset" "error_count" {
  workspace = data.observe_workspace.default.oid
  name      = "Error Count Metric"

  metric_name = "error_count"
  metric_type = "gauge"
  unit        = "errors"
  interval    = "1m"

  input_dataset = observe_dataset.logs.oid
  shaping_query = "filter severity = \"ERROR\""

  aggregation {
    function = "count"
  }

  metric_tags {
    name         = "service"
    field_column = "service_name"
    field_path   = "."
  }

  metric_tags {
    name         = "environment"
    field_column = "env"
    field_path   = "."
  }
}

# Log derived metric with sum aggregation on a field
resource "observe_log_derived_metric_dataset" "total_bytes" {
  workspace   = data.observe_workspace.default.oid
  name        = "Total Bytes Transferred"
  description = "Sum of bytes transferred per service"

  metric_name = "bytes_transferred"
  metric_type = "counter"
  unit        = "bytes"
  interval    = "5m"

  input_dataset = observe_dataset.access_logs.oid
  shaping_query = "filter status_code >= 200 and status_code < 300"

  aggregation {
    function     = "sum"
    field_column = "bytes_sent"
    field_path   = "."
  }

  metric_tags {
    name         = "service"
    field_column = "service_name"
    field_path   = "."
  }
}

# Log derived metric with average aggregation
resource "observe_log_derived_metric_dataset" "avg_response_time" {
  workspace = data.observe_workspace.default.oid
  name      = "Average Response Time"

  metric_name = "response_time_avg"
  metric_type = "gauge"
  unit        = "milliseconds"
  interval    = "1m"

  input_dataset = observe_dataset.api_logs.oid
  shaping_query = "filter endpoint != \"/health\""

  aggregation {
    function     = "avg"
    field_column = "response_time_ms"
    field_path   = "."
  }

  metric_tags {
    name         = "endpoint"
    field_column = "endpoint"
    field_path   = "."
  }

  metric_tags {
    name         = "method"
    field_column = "http_method"
    field_path   = "."
  }
}

# Log derived metric with count_distinct aggregation
resource "observe_log_derived_metric_dataset" "unique_users" {
  workspace = data.observe_workspace.default.oid
  name      = "Unique Active Users"

  metric_name = "active_users"
  metric_type = "gauge"
  unit        = "users"
  interval    = "10m"

  input_dataset = observe_dataset.user_activity.oid
  shaping_query = "filter action = \"login\""

  aggregation {
    function     = "count_distinct"
    field_column = "user_id"
    field_path   = "."
  }

  metric_tags {
    name         = "region"
    field_column = "region"
    field_path   = "."
  }
}

