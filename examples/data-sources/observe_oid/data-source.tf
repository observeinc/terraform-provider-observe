data "observe_workspace" "default" {
  name = "Default"
}


# parse an OID into components
data "observe_oid" "workspace" {
  oid = data.observe_workspace.default.oid
}

# construct an OID from components
data "observe_oid" "workspace" {
  id = data.observe_workspace.default.id
  type = "workspace"
  version = "1"
}
