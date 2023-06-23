locals {
  repository = "observeinc/terraform-provider-observe"
}

resource "github_actions_secret" "observe_provider_password" {
  repository  = local.repository
  secret_name = "OBSERVE_USER_PASSWORD"

  # To generate this value:
  # gh secret set --no-store OBSERVE_USER_PASSWORD
  encrypted_value = "9WqMEnZeoJFvJVN5jwnXMxV40jixuGlrJHj96J12L1M06ByimWK1GpKFwTfU+05nDt98Z7PRJf6DaaIGi8i1LLOq5g=="
}

resource "github_actions_variable" "observe_provider" {
  for_each = {
    OBSERVE_CUSTOMER  = "127814973959"
    OBSERVE_DOMAIN    = "observe-eng.com"
    OBSERVE_USER      = "github-terraform-provider@observeinc.com"
    OBSERVE_WORKSPACE = "Kubernetes"
  }

  repository    = local.repository
  variable_name = each.key
  value         = each.value
}
