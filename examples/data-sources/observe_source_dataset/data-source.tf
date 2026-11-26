data "observe_source_dataset" "logs" {
  label = "myapp/Logs"
}

output "logs_storage" {
  value = data.observe_source_dataset.logs.storage
}

output "logs_retention_days" {
  value = data.observe_source_dataset.logs.effective_data_retention_days
}
