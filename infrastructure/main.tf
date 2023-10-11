locals {
  organization = "observeinc"
  repository   = "terraform-provider-observe"
}

resource "github_actions_secret" "secrets" {
  for_each = setsubtract(fileset("${path.module}/secrets", "*"), ["README.md"])

  repository  = local.repository
  secret_name = each.key

  encrypted_value = file("${path.module}/secrets/${each.key}")
}

resource "github_dependabot_secret" "secrets" {
  for_each = {
    # Automatically expose any OBSERVE_* credentials as Dependabot secrets to allow aceptance testing PRs
    for k, v in github_actions_secret.secrets : k => v if startswith(k, "OBSERVE")
  }

  repository      = each.value.repository
  secret_name     = each.key
  encrypted_value = each.value.encrypted_value
}

moved {
  from = github_actions_secret.observe_provider_password
  to   = github_actions_secret.secrets["OBSERVE_PROVIDER_PASSWORD"]
}

resource "github_actions_variable" "observe_provider" {
  for_each = {
    OBSERVE_CUSTOMER   = "127814973959"
    OBSERVE_DOMAIN     = "observe-eng.com"
    OBSERVE_USER_EMAIL = "github-terraform-provider@observeinc.com"
    OBSERVE_WORKSPACE  = "Kubernetes"
  }

  repository    = local.repository
  variable_name = each.key
  value         = each.value
}

resource "github_actions_variable" "observe_filedrop_role_arn" {
  repository    = local.repository
  variable_name = "OBSERVE_FILEDROP_ROLE_ARN"
  value         = "arn:aws:iam::723346149663:role/jyc-snowpipe-assume-role"
}
