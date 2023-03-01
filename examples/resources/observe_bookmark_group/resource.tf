data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_bookmark_group" "example" {
  workspace 	 = data.observe_workspace.default.oid
  name      	 = "Example"
}
