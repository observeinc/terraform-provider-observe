data "observe_inbound_share_setup" "setup" {}

# Grant the share to Observe's consumer account
output "observe_consumer_account" {
  value = data.observe_inbound_share_setup.setup.consumer_account
}

# SQL to run in the provider account, for shares not managed by Terraform
output "setup_sql" {
  value = [for step in data.observe_inbound_share_setup.setup.setup_sql : step.sql]
}
