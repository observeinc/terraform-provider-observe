data "observe_inbound_share_setup" "setup" {}

output "observe_consumer_account" {
  description = "Observe's Snowflake consumer account, to add as a recipient of the share."
  value       = data.observe_inbound_share_setup.setup.consumer_account
}

output "setup_sql" {
  description = "SQL to run in the provider account, for shares not managed by Terraform."
  value       = [for step in data.observe_inbound_share_setup.setup.setup_sql : step.sql]
}
