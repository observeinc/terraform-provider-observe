data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "usage_metrics" {
  workspace = data.observe_workspace.default.oid
  name      = "usage/Observe Usage Metrics"
}

# threshold type monitor
resource "observe_monitor_v2" "example" {
  data_stabilization_delay = "0s"
  description              = "Credit usage monitor"
  inputs = {
    "credits_adhoc_query_from_usage/Observe Usage Metrics" = data.observe_dataset.usage_metrics.oid
  }
  lookback_time = "10m0s"
  name          = "example"
  rule_kind     = "threshold"
  workspace     = data.observe_workspace.default.oid
  groupings {
    link_column {
      name = "Dashboard"
    }
  }

  no_data_rules {
    threshold {
      value_column_name = "A_credits_adhoc_query_sum"
      aggregation       = "all_of"
    }
  }

  rules {
    level = "error"
    threshold {
      aggregation       = "all_of"
      value_column_name = "A_credits_adhoc_query_sum"

      compare_values {
        compare_fn = "greater"
        value_float64 = [
          100,
        ]
      }
    }
  }

  stage {
    output_stage = false
    pipeline     = <<-EOT
      align 1m, frame(back: 1h), A_credits_adhoc_query_sum:sum(m("credits_adhoc_query"))
      aggregate A_credits_adhoc_query_sum:sum(A_credits_adhoc_query_sum), group_by(^Dashboard...)
    EOT
  }
}

# anomaly type monitor
resource "observe_monitor_v2" "anomaly_example" {
  name          = "Anomaly Monitor Example"
  workspace     = data.observe_workspace.default.oid
  rule_kind     = "anomaly"
  lookback_time = "10m0s"
  inputs = {
    "credits_adhoc_query_from_usage/Observe Usage Metrics" = data.observe_dataset.usage_metrics.oid
  }

  stage {
    pipeline = <<-EOT
      align 2m, frame(back: 2m), A_credits_adhoc_query_sum:sum(m("credits_adhoc_query"))/to_minutes(bin_size())
      aggregate A_credits_adhoc_query_sum:sum(A_credits_adhoc_query_sum), group_by(^Dataset...)
    EOT
  }

  groupings {
    link_column {
      name = "Dataset"
    }
  }

  rule_template {
    anomaly {
      value_column_name       = "A_credits_adhoc_query_sum"
      compare_fn              = "above"
      num_standard_deviations = 3
      basic_algorithm {}
    }
  }

  rules {
    level = "informational"
    anomaly {
      compare_percentage = 50
    }
  }

  rules {
    level = "warning"
    anomaly {
      compare_percentage = 75
    }
  }
}
