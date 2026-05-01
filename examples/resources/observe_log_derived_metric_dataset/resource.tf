terraform {
  required_providers {
    observe = {
      source = "observeinc/observe"
    }
  }
}

data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "default" {
  workspace = data.observe_workspace.default.oid
  name      = "Default"
}

# Simplest possible log-derived metric: counts all Prometheus observations
resource "observe_log_derived_metric_dataset" "simple_count" {
  workspace = data.observe_workspace.default.oid

  metric_name = "prometheus_observation_count"

  input         = data.observe_datastream.default.dataset
  shaping_query = "filter OBSERVATION_KIND = \"prometheus\""

  aggregation {
    function = "count"
  }
}

# Log-derived metric counting non-200 HTTP handler requests per job
resource "observe_log_derived_metric_dataset" "error_count" {
  workspace   = data.observe_workspace.default.oid
  description = "Counts non-200 promhttp handler requests per job"

  metric_name = "http_error_request_count"
  metric_type = "gauge"
  unit        = "1"
  interval    = "1m"

  input         = data.observe_datastream.default.dataset
  shaping_query = <<-EOT
    filter OBSERVATION_KIND = "prometheus"
    make_col metric_name:string(EXTRA.__name__),
             response_code:string(EXTRA.code),
             job_name:string(EXTRA.job)
    filter metric_name = "promhttp_metric_handler_requests_total"
    filter response_code != "200"
  EOT

  aggregation {
    function = "count"
  }

  metric_tag {
    name   = "job"
    column = "job_name"
  }

  metric_tag {
    name   = "code"
    column = "response_code"
  }
}

# Log-derived metric summing network bytes transmitted per job and namespace
resource "observe_log_derived_metric_dataset" "total_bytes" {
  workspace   = data.observe_workspace.default.oid
  description = "Sum of network bytes transmitted per job and namespace"

  metric_name = "network_bytes_transmitted"
  metric_type = "cumulative_counter"
  unit        = "bytes"
  interval    = "5m"

  input         = data.observe_datastream.default.dataset
  shaping_query = <<-EOT
    filter OBSERVATION_KIND = "prometheus"
    make_col metric_name:string(EXTRA.__name__),
             metric_value:float64(FIELDS.value),
             job_name:string(EXTRA.job),
             k8s_namespace:string(EXTRA.k8s_namespace_name)
    filter metric_name = "process_network_transmit_bytes_total"
  EOT

  aggregation {
    function = "sum"
    field_path {
      column = "metric_value"
    }
  }

  metric_tag {
    name   = "job"
    column = "job_name"
  }

  metric_tag {
    name   = "namespace"
    column = "k8s_namespace"
  }
}

# Log-derived metric averaging GC duration (p50) per job and namespace
resource "observe_log_derived_metric_dataset" "avg_gc_duration" {
  workspace = data.observe_workspace.default.oid

  metric_name = "gc_duration_p50_avg"
  metric_type = "gauge"
  unit        = "seconds"
  interval    = "1m"

  input         = data.observe_datastream.default.dataset
  shaping_query = <<-EOT
    filter OBSERVATION_KIND = "prometheus"
    make_col metric_name:string(EXTRA.__name__),
             metric_value:float64(FIELDS.value),
             quantile_label:string(EXTRA.quantile),
             job_name:string(EXTRA.job),
             k8s_namespace:string(EXTRA.k8s_namespace_name)
    filter metric_name = "go_gc_duration_seconds"
    filter quantile_label = "0.5"
  EOT

  aggregation {
    function = "avg"
    field_path {
      column = "metric_value"
    }
  }

  metric_tag {
    name   = "job"
    column = "job_name"
  }

  metric_tag {
    name   = "namespace"
    column = "k8s_namespace"
  }
}

# Log-derived metric counting distinct pods per namespace and job
resource "observe_log_derived_metric_dataset" "unique_pods" {
  workspace = data.observe_workspace.default.oid

  metric_name = "active_pods"
  metric_type = "gauge"
  unit        = "pods"
  interval    = "10m"

  input         = data.observe_datastream.default.dataset
  shaping_query = <<-EOT
    filter OBSERVATION_KIND = "prometheus"
    make_col pod_name:string(EXTRA.k8s_pod_name),
             k8s_namespace:string(EXTRA.k8s_namespace_name),
             job_name:string(EXTRA.job)
    filter not is_null(pod_name)
  EOT

  aggregation {
    function = "count_distinct"
    field_path {
      column = "pod_name"
    }
  }

  metric_tag {
    name   = "namespace"
    column = "k8s_namespace"
  }

  metric_tag {
    name   = "job"
    column = "job_name"
  }
}
