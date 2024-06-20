terraform {
  required_providers {
    observe = {
      source  = "observeinc/observe"
      version = "~> 0.14"
    }
  }
}

# Configure the observe provider
provider "observe" {}

# Look up existing workspace 
data "observe_workspace" "default" {
  name = "Default"
}
