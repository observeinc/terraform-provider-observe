data "observe_app_version" "kubernetes" {
  module_id          = "observeinc/kubernetes/observe"
  version_constraint = ">= 1, < 2"
}

resource "observe_app" "kubernetes" {
  module_id = data.observe_app_version.kubernetes.module_id
  version   = data.observe_app_version.kubernetes.version

  # ...
}
