data "observe_workspace" "default" {
  name = "Default"
}

resource "observe_folder" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Folder"
}

# Reference the folder from other resources that support folders:
resource "observe_app" "example" {
  folder = observe_folder.example.oid
  # ...
}
