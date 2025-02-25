data "observe_workspace" "default" {
  name = "Default"
}

data "observe_datastream" "example" {
  workspace = data.observe_workspace.default.oid
  name      = "My Datastream"
}

# this creates the filedrop in the Observe service. The AWS bucket and forwarders will
# still need be configured using the observe forwarder module using the following
# computed attributes from this resource:
# - observe_filedrop.example.endpoint.0.s3.0.arn
# - observe_filedrop.example.endpoint.0.s3.0.bucket
# - observe_filedrop.example.endpoint.0.s3.0.prefix
resource "observe_filedrop" "example" {
  workspace  = data.observe_workspace.default.oid
  datastream = data.observe_datastream.example.oid
  config {
    provider {
      aws {
        region  = "us-west-2"
        role_arn = "arn:aws:iam:<account>:role/<myrole>"
      }
    }
  }
}
