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

  repositor     = local.repository
  variable_name = each.key
  value         = each.value
}
