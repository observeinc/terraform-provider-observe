data "observe_rbac_group" "example" {
  name = "example"
}

// In RBAC v2, "everyone" is a special pre-defined group that always includes all users.
// Reach out to Observe to enable this feature.
data "observe_rbac_group" "everyone" {
  name = "everyone"
}
