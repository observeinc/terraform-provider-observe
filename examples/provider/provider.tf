provider "observe" {
  customer      = var.customer      # optionally use OBSERVE_CUSTOMER env var
  user_email    = var.user_email    # optionally use OBSERVE_USER_EMAIL env var
  user_password = var.user_password # optionally use OBSERVE_USER_PASSWORD env var
}
