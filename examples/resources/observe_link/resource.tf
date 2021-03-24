data "observe_workspace" "default" {
  name = "Default"
}

data "observe_dataset" "container" {
  workspace = data.observe_workspace.default.oid
  name      = "Container"
}

data "observe_dataset" "node" {
  workspace = data.observe_workspace.default.oid
  name      = "Node"
}

resource "observe_link" "container_to_node" {
  workspace = data.observe_workspace.default.oid

  source = data.observe_dataset.container.oid
  target = data.observe_dataset.node.oid

  /* The container dataset has `clusterUid` and `nodeName` as columns,
   * whereas the node dataset has declared `clusterUid` and `name` as a key.
   */
  fields = ["clusterUid", "nodeName:name"]
}
