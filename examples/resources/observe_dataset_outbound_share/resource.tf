data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "observation" {
  workspace = data.observe_workspace.default.oid
  name      = "Observation"
}

# dataset outbound shares need a valid backing snowflake outbound
resource "observe_snowflake_outbound_share" "example" {
  workspace   = data.observe_workspace.default.oid
  name        = "Example SF Outbound"
  description = "Example description"

  account {
    account      = "my_sf_acct"
    organization = "my_sf_org"
  }
}

resource "observe_dataset_outbound_share" "example" {
  workspace      = data.observe_workspace.default.oid
  description    = "Example description"
  name           = "Example outbound share"
  dataset        = data.observe_dataset.observation.oid
  outbound_share = observe_snowflake_outbound_share.example.oid
  schema_name    = "example_schema"
  view_name      = "example_view"
  freshness_goal = "15m"
}
